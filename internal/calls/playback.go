/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package calls

import (
	"ashokshau/tgmusic/internal/cache"
	"ashokshau/tgmusic/internal/db"
	"ashokshau/tgmusic/internal/disk"
	"ashokshau/tgmusic/internal/downloader"
	"ashokshau/tgmusic/internal/utils"
	"context"
	"errors"
	"fmt"
	"html"
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
	case strings.Contains(msg, "GROUPCALL_ADD_PARTICIPANTS_FAILED"),
		strings.Contains(msg, "Timeout while fetching data"),
		strings.Contains(msg, "INTERDC_X_CALL_ERROR"):
		return errRetryOnce
	case strings.Contains(msg, "CHANNELS_TOO_MUCH"),
		strings.Contains(msg, "FROZEN_METHOD_INVALID"),
		strings.Contains(msg, "FLOOD_WAIT_"),
		strings.Contains(msg, "FLOOD_PREMIUM_WAIT_"),
		strings.Contains(msg, "limiting join attempts"),
		strings.Contains(msg, "USER_DEACTIVATED"):
		return errRotate
	default:
		return errUnknown
	}
}

func (c *TelegramCalls) playMediaWithAssistant(bot *td.Client, chatID int64, filePath string, video bool, ffmpegParameters string, call *AssistantAccount, index int) error {
	if chatID > 0 {
		return fmt.Errorf("Playback is only available in groups.")
	}

	if err := c.ensureAssistantMember(bot, chatID, call, index); err != nil {
		return err
	}

	logger.Debug("Playing media in chat", "id", chatID, "path", filePath, "index", index)

	if ffmpegParameters == "" {
		c.clearPlayedTimeOffset(chatID)
	}

	mediaDesc := getMediaDescription(filePath, video, chatID, ffmpegParameters)
	if err := c.startCallStream(context.Background(), call, chatID, mediaDesc); err != nil {
		return err
	}

	if db.Instance.GetLoggerStatus() {
		go sendLogger(bot, chatID, cache.ChatCache.GetPlayingTrack(chatID))
	}

	return nil
}

func (c *TelegramCalls) PlayMedia(bot *td.Client, chatID int64, filePath string, video bool, ffmpegParameters ...string) (err error) {
	release := disk.Hold(filePath)
	defer func() {
		if err != nil {
			cache.ChatCache.ClearChat(chatID)
		} else {
			disk.SetPlaying(chatID, filePath)
		}
		release()
	}()

	ffParam := ""
	if len(ffmpegParameters) > 0 {
		ffParam = ffmpegParameters[0]
	}

	if chatID > 0 {
		return fmt.Errorf("Playback is only available in groups.")
	}
	call, index, err := c.GetAccount(chatID)
	if err != nil {
		logger.Errorf("get playback assistant for chat %d: %v", chatID, err)
		return fmt.Errorf("No playback assistant is available right now. Please try again later.")
	}
	err = c.playWithAssistants(chatID, call, index, playbackActions{
		attempt: func(call *AssistantAccount, index int) error {
			return c.playMediaWithAssistant(bot, chatID, filePath, video, ffParam, call, index)
		},
		save: func(index int) {
			if err := db.Instance.SetAssistant(chatID, index); err != nil {
				logger.Errorf("save assistant %d for chat %d: %v", index, chatID, err)
			}
		},
		evict: func(index int, err error) {
			c.evictAssistant(chatID, index, err)
		},
	})
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "groupcall_invalid"), strings.Contains(message, "no active group call"):
		return fmt.Errorf("<b>Voice chat is unavailable.</b> Start a new voice chat and try again.")
	case strings.Contains(message, "is closed"), strings.Contains(message, "groupcall_forbidden"):
		return fmt.Errorf("<b>No active video chat found.</b> Please start one and try again.")
	case strings.Contains(message, "flood_wait"), strings.Contains(message, "limiting join attempts"):
		return fmt.Errorf("Telegram is temporarily limiting assistant joins. Please try again later.")
	case strings.Contains(message, "channels_too_much"):
		return fmt.Errorf("The assistant has joined too many channels or supergroups. No available assistant could start playback. Please try again later.")
	case strings.Contains(message, "invite link"), strings.Contains(message, "invite_hash"):
		return fmt.Errorf("The assistant invite link is invalid or expired. Please run /reload and try again.")
	case strings.Contains(message, "banned"), strings.Contains(message, "restricted"), strings.Contains(message, "admin"):
		return fmt.Errorf("The assistant cannot join. Check that it is not banned and the bot has the required admin permissions.")
	default:
		return fmt.Errorf("Unable to start playback. Please try again.")
	}
}

func (c *TelegramCalls) evictAssistant(chatID int64, index int, err error) {
	_ = db.Instance.RemoveAssistant(chatID)
	if strings.Contains(err.Error(), "CHANNELS_TOO_MUCH") {
		go func() { _, _ = c.LeaveAllForAccount(index) }()
	}
}

type playbackActions struct {
	attempt func(*AssistantAccount, int) error
	save    func(int)
	evict   func(int, error)
}

// playWithAssistants tries each assistant once, retrying transient call errors
// once on the same account before rotating.
func (c *TelegramCalls) playWithAssistants(chatID int64, call *AssistantAccount, index int, actions playbackActions) error {
	tried := make(map[int]bool)
	for {
		tried[index] = true
		err := actions.attempt(call, index)
		if err == nil {
			actions.save(index)
			return nil
		}
		logger.Warnf("start playback in chat %d with assistant %d: %v", chatID, index, err)

		if classifyError(err) == errRetryOnce {
			err = actions.attempt(call, index)
			if err == nil {
				actions.save(index)
				return nil
			}
			logger.Warnf("retry playback in chat %d with assistant %d: %v", chatID, index, err)
		}
		if classifyError(err) != errRotate {
			return err
		}
		actions.evict(index, err)

		nextCall, nextIndex, nextErr := c.nextUntried(tried)
		if nextErr != nil {
			logger.Errorf("find another playback assistant in chat %d: %v; last playback error: %v", chatID, nextErr, err)
			return err
		}
		logger.Infof("retrying playback in chat %d with assistant %d after assistant %d failed", chatID, nextIndex, index)
		call, index = nextCall, nextIndex
	}
}

func (c *TelegramCalls) prepareTrackDownload(bot *td.Client, song *utils.PlayerCache, reply *td.Message) error {
	if song.FilePath != "" {
		return nil
	}

	dlPath, err := downloader.DlCachedTrack(song, bot, nil)
	song.FilePath = dlPath
	if err == nil && song.FilePath == "" {
		err = errors.New("download returned no file")
	}
	if err != nil {
		logger.Warnf("download queued track for chat %d: %v", reply.ChatId, err)
		if _, editErr := reply.EditText(bot, "⚠️ Download failed. Skipping track...", nil); editErr != nil {
			logger.Warnf("send queued download failure in chat %d: %v", reply.ChatId, editErr)
		}
		return err
	}

	return nil
}

func (c *TelegramCalls) PlayNext(bot *td.Client, chatID int64) error {
	loop := cache.ChatCache.GetLoopCount(chatID)
	if loop > 0 {
		cache.ChatCache.SetLoopCount(chatID, loop-1)
		if currentsSong := cache.ChatCache.GetPlayingTrack(chatID); currentsSong != nil {
			return c.playTrack(bot, chatID, currentsSong)
		}
	}

	finishedTrack := cache.ChatCache.RemoveCurrentSong(chatID)
	if nextSong := cache.ChatCache.GetPlayingTrack(chatID); nextSong != nil {
		return c.playTrack(bot, chatID, nextSong)
	}

	lastTrack := cache.ChatCache.GetLastAutoplayTrackID(chatID)
	if finishedTrack != nil {
		if finishedTrack.Platform == utils.Telegram || finishedTrack.Platform == utils.DirectLink {
			lastTrack = strings.TrimSpace(finishedTrack.Name + " " + finishedTrack.Channel)
		}
	}
	if lastTrack != "" && cache.ChatCache.GetAutoplay(chatID) {
		return c.handleAutoplay(bot, chatID, lastTrack)
	}

	return c.handleNoSong(bot, chatID)
}

func (c *TelegramCalls) handleNoSong(bot *td.Client, chatID int64) error {
	_ = c.Stop(chatID, false)
	_, sendErr := bot.SendMessage(chatID, "🎵 Queue finished. Add more songs with /play.", nil)
	if sendErr != nil {
		logger.Warnf("send queue finished in chat %d: %v", chatID, sendErr)
	}
	return sendErr
}

func (c *TelegramCalls) playTrack(bot *td.Client, chatID int64, song *utils.PlayerCache) error {
	reply, err := bot.SendMessage(chatID, fmt.Sprintf("Downloading %s...", song.Name), nil)
	if err != nil {
		logger.Warnf("send download progress in chat %d: %v", chatID, err)
		return err
	}

	return c.PlayTrackWithMessage(bot, reply, chatID, song)
}

// PlayTrackWithMessage starts the queued track using an existing progress message.
func (c *TelegramCalls) PlayTrackWithMessage(bot *td.Client, reply *td.Message, chatID int64, song *utils.PlayerCache) error {
	if err := c.prepareTrackDownload(bot, song, reply); err != nil {
		return c.PlayNext(bot, chatID)
	}

	if err := c.PlayMedia(bot, chatID, song.FilePath, song.IsVideo); err != nil {
		if _, editErr := reply.EditText(bot, err.Error(), &td.EditMessageTextOpts{ParseMode: "HTML", DisableWebPagePreview: true}); editErr != nil {
			logger.Warnf("send playback failure in chat %d: %v", chatID, editErr)
		}
		return nil
	}

	if song.Duration == 0 {
		song.Duration = utils.GetMediaDuration(song.FilePath)
	}

	text := fmt.Sprintf(
		"<u><b>| Started streaming</b></u>\n\n<b>Title:</b> <a href='%s'>%s</a>\n\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
		html.EscapeString(song.URL),
		html.EscapeString(song.Name),
		utils.SecToMin(song.Duration),
		html.EscapeString(song.User),
	)

	_, err := reply.EditText(bot, text, &td.EditMessageTextOpts{
		ReplyMarkup:           utils.ControlButtons("play"),
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})

	if err != nil && !strings.Contains(err.Error(), "MESSAGE_NOT_MODIFIED") {
		logger.Warnf("edit playback progress in chat %d: %v", chatID, err)
	}

	return nil
}
