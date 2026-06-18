package vc

import (
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"context"
	"errors"
	"fmt"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

// errorKind classifies a Telegram group call error for retry strategy.
type errorKind int

const (
	errFatal     errorKind = iota // return immediately with a user-facing message
	errRetryOnce                  // retry the same assistant once (e.g. participants race)
	errRotate                     // try a different assistant (flood/frozen/channels)
	errUnknown                    // log and return as-is
)

func classifyError(err error) errorKind {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "is closed"),
		strings.Contains(msg, "GROUPCALL_FORBIDDEN"):
		return errFatal
	case strings.Contains(msg, "GROUPCALL_INVALID"):
		return errFatal
	case strings.Contains(msg, "GROUPCALL_ADD_PARTICIPANTS_FAILED"):
		return errRetryOnce
	case strings.Contains(msg, "CHANNELS_TOO_MUCH"),
		strings.Contains(msg, "FROZEN_METHOD_INVALID"),
		strings.Contains(msg, "FLOOD_WAIT_X"):
		return errRotate
	default:
		return errUnknown
	}
}

func fatalMessage(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "is closed") || strings.Contains(msg, "GROUPCALL_FORBIDDEN") {
		return errors.New("<b>No active video chat found.</b>\n\nPlease start one and <b>try again</b>")
	}

	if strings.Contains(msg, "GROUPCALL_INVALID") {
		return errors.New("<b>GROUPCALL_INVALID:</b> start a video chat and try again.\n\nIf the problem persists, please report it to the developer.")
	}
	return err
}

// PlayMedia plays media in a voice chat with automatic assistant rotation on certain errors.
func (c *TelegramCalls) PlayMedia(bot *td.Client, chatID int64, filePath string, video bool, ffmpegParameters string) error {
	call, index, err := c.GetGroupAssistant(chatID)
	if err != nil {
		return err
	}

	err = c.playMedia(bot, chatID, filePath, video, ffmpegParameters, call, index)
	if err == nil {
		_ = db.Instance.SetAssistant(chatID, index)
		return nil
	}

	switch classifyError(err) {
	case errFatal:
		return fatalMessage(err)

	case errUnknown:
		logger.Error("Failed to play the media", "error", err, "index", index, "chatID", chatID)
		return fmt.Errorf("playback failed: %w", err)

	case errRetryOnce:
		err = c.playMedia(bot, chatID, filePath, video, ffmpegParameters, call, index)
		if err == nil {
			_ = db.Instance.SetAssistant(chatID, index)
			return nil
		}
		if classifyError(err) != errRotate {
			return fmt.Errorf("playback failed: %w", err)
		}
		fallthrough // GROUPCALL_ADD_PARTICIPANTS_FAILED can escalate to rotation

	case errRotate:
		c.evictAssistant(chatID, index, err)
	}

	return c.rotateAndPlay(bot, chatID, filePath, video, ffmpegParameters, map[int]bool{index: true}, err)
}

// evictAssistant cleans up state for an assistant that can no longer serve a chat.
func (c *TelegramCalls) evictAssistant(chatID int64, index int, err error) {
	_ = db.Instance.RemoveAssistant(chatID)
	if strings.Contains(err.Error(), "CHANNELS_TOO_MUCH") {
		go func() { _, _ = c.LeaveAllForClient(index) }()
	}
}

func (c *TelegramCalls) playMedia(bot *td.Client, chatID int64, filePath string, video bool, ffmpegParameters string, call *Assistant, index int) error {
	if chatID > 0 {
		return errors.New("private calls are not supported for media playback")
	}

	if err := c.joinAssistant(bot, chatID, call, index); err != nil {
		cache.ChatCache.ClearChat(chatID)
		return err
	}

	logger.Debug("Playing media in chat", "id", chatID, "path", filePath, "index", index)

	mediaDesc := getMediaDescription(filePath, video, ffmpegParameters)
	if err := call.Play(context.Background(), chatID, mediaDesc); err != nil {
		cache.ChatCache.ClearChat(chatID)
		return err
	}

	if db.Instance.GetLoggerStatus() {
		go sendLogger(bot, chatID, cache.ChatCache.GetPlayingTrack(chatID))
	}

	return nil
}

// rotateAndPlay iterates over all remaining assistants until one succeeds or all are exhausted.
func (c *TelegramCalls) rotateAndPlay(bot *td.Client, chatID int64, filePath string, video bool, ffmpegParameters string, tried map[int]bool, lastErr error) error {
	for {
		call, nextIndex, err := c.nextUntried(tried)
		if err != nil {
			logger.Error("Playback failed after full rotation", "error", lastErr, "chatID", chatID)
			return fmt.Errorf("playback failed after trying all assistants: %w", lastErr)
		}
		tried[nextIndex] = true

		err = c.playMedia(bot, chatID, filePath, video, ffmpegParameters, call, nextIndex)
		if err == nil {
			_ = db.Instance.SetAssistant(chatID, nextIndex)
			return nil
		}
		lastErr = err

		switch classifyError(err) {
		case errRetryOnce:
			err = c.playMedia(bot, chatID, filePath, video, ffmpegParameters, call, nextIndex)
			if err == nil {
				_ = db.Instance.SetAssistant(chatID, nextIndex)
				return nil
			}
			lastErr = err
			if classifyError(err) == errRotate {
				c.evictAssistant(chatID, nextIndex, err)
				continue
			}
			return fmt.Errorf("playback failed: %w", lastErr)

		case errRotate:
			c.evictAssistant(chatID, nextIndex, err)
			continue

		default:
			// errFatal or errUnknown — stop rotating.
			return fmt.Errorf("playback failed: %w", lastErr)
		}
	}
}

// nextUntried finds the next assistant index not yet tried in this rotation round.
func (c *TelegramCalls) nextUntried(tried map[int]bool) (*Assistant, int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for i, call := range c.assistants {
		if !tried[i] {
			return call, i, nil
		}
	}
	return nil, -1, errors.New("no untried assistants remain")
}
