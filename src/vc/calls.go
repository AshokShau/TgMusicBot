/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package vc

/*
#cgo linux LDFLAGS: -L . -lntgcalls -lm -lz
#cgo darwin LDFLAGS: -L . -lntgcalls -lc++ -lz -lbz2 -liconv -framework AVFoundation -framework AudioToolbox -framework CoreAudio -framework QuartzCore -framework CoreMedia -framework VideoToolbox -framework AppKit -framework Metal -framework MetalKit -framework OpenGL -framework IOSurface -framework ScreenCaptureKit

// Currently is supported only dynamically linked library on Windows due to
// https://github.com/golang/go/issues/63903
#cgo windows LDFLAGS: -L. -lntgcalls
#include "ntgcalls/ntgcalls.h"
#include "glibc_compatibility.h"
*/
import "C"

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/utils"
	"ashokshau/tgmusic/src/vc/ntgcalls"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"math/big"
	"os"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

// getClientIndex selects an assistant client index (0-based) for a given chat.
func (c *TelegramCalls) getClientIndex(chatID int64) (int, error) {
	c.mu.RLock()
	totalClients := len(c.assistants)
	c.mu.RUnlock()

	if totalClients == 0 {
		return -1, fmt.Errorf("no clients are available")
	}

	assignedIndex, err := db.Instance.GetAssistant(chatID)
	if err != nil {
		slog.Info("[TelegramCalls] DB.GetAssistant error", "error", err)
		assignedIndex = -1
	}

	if assignedIndex >= 0 && assignedIndex < totalClients {
		return assignedIndex, nil
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(totalClients)))
	if err != nil {
		slog.Info("[TelegramCalls] Could not generate a random number", "error", err)
		newClientIndex := 0
		if assignedIndex == -1 && chatID != 0 {
			if _, err := db.Instance.AssignAssistant(chatID, newClientIndex); err != nil {
				logger.Info("[TelegramCalls] DB.AssignAssistant error", "error", err)
			}
		}
		return newClientIndex, nil
	}

	newClientIndex := int(n.Int64())
	if chatID != 0 {
		if _, err := db.Instance.AssignAssistant(chatID, newClientIndex); err != nil {
			logger.Info("[TelegramCalls] DB.AssignAssistant error", "error", err)
		}
	}

	return newClientIndex, nil
}

// GetGroupAssistant retrieves the assistant and its index for a given chat.
func (c *TelegramCalls) GetGroupAssistant(chatID int64) (*Assistant, int, error) {
	clientIndex, err := c.getClientIndex(chatID)
	if err != nil {
		return nil, -1, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	call, ok := c.assistants[clientIndex]
	if !ok {
		return nil, -1, fmt.Errorf("no ntgcalls instance was found for client index %d", clientIndex)
	}
	return call, clientIndex, nil
}

func (c *TelegramCalls) playMedia(chatID int64, filePath string, video bool, ffmpegParameters string, call *Assistant, index int) error {
	if chatID > 0 {
		return errors.New("private calls are not supported for media playback")
	}

	if err := c.joinAssistant(chatID, call, index); err != nil {
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
		go sendLogger(c.bot, chatID, cache.ChatCache.GetPlayingTrack(chatID))
	}

	return nil
}

// PlayMedia plays media in a voice chat with automatic assistant rotation on certain errors.
func (c *TelegramCalls) PlayMedia(chatID int64, filePath string, video bool, ffmpegParameters string) error {
	tried := make(map[int]bool)
	var lastErr error

	for {
		c.mu.RLock()
		totalClients := len(c.assistants)
		c.mu.RUnlock()

		if len(tried) >= totalClients {
			if lastErr != nil {
				return lastErr
			}
			return fmt.Errorf("no available assistants to play media")
		}

		call, index, err := c.GetGroupAssistant(chatID)
		if err != nil {
			return err
		}

		if tried[index] {
			index = -1
			c.mu.RLock()
			for i, ctx := range c.assistants {
				if !tried[i] {
					index = i
					call = ctx
					break
				}
			}
			c.mu.RUnlock()
			if index == -1 {
				break
			}
		}

		tried[index] = true

		err = c.playMedia(chatID, filePath, video, ffmpegParameters, call, index)
		if err == nil {
			_ = db.Instance.SetAssistant(chatID, index)
			return nil
		}

		lastErr = err

		if strings.Contains(err.Error(), "is closed") || strings.Contains(err.Error(), "GROUPCALL_FORBIDDEN") {
			return errors.New("<b>No active video chat found.</b>\n\nPlease start one and <b>try again</b>")
		}

		if strings.Contains(err.Error(), "GROUPCALL_INVALID") {
			return fmt.Errorf("<b>GROUPCALL_INVALID:</b> start a video chat and try again.\n\nIf the problem persists, please report it to the developer.")
		}

		if strings.Contains(err.Error(), "CHANNELS_TOO_MUCH") {
			go func(idx int) {
				_, _ = c.LeaveAllForClient(idx)
			}(index)

			_ = db.Instance.RemoveAssistant(chatID)
			continue
		}

		if strings.Contains(err.Error(), "FROZEN_METHOD_INVALID") || strings.Contains(err.Error(), "FLOOD_WAIT_X") {
			_ = db.Instance.RemoveAssistant(chatID)
			continue
		}

		logger.Error("Failed to play the media", "error", err, "index", index, "chatID", chatID)
		return fmt.Errorf("playback failed: %w", err)
	}

	return fmt.Errorf("failed to play media after trying all assistants: %w", lastErr)
}

// playSong downloads and plays a single song. It sends a message to the chat to indicate the download status
// and updates it with the song's information once playback begins.
func (c *TelegramCalls) playSong(chatID int64, song *utils.CachedTrack) error {
	reply, err := c.bot.SendTextMessage(chatID, fmt.Sprintf("Downloading %s...", song.Name), nil)
	if err != nil {
		slog.Info("[playSong] Failed to send message", "error", err)
		return err
	}

	if err = c.downloadAndPrepareSong(song, reply); err != nil {
		return c.PlayNext(chatID)
	}

	if err = c.PlayMedia(chatID, song.FilePath, song.IsVideo, ""); err != nil {
		_, _ = reply.EditText(c.bot, err.Error(), &td.EditTextMessageOpts{ParseMode: "HTML", DisableWebPagePreview: true})
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

	_, err = reply.EditText(c.bot, text, &td.EditTextMessageOpts{
		ReplyMarkup:           core.ControlButtons("play"),
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})

	if err != nil {
		slog.Info("[playSong] Failed to edit message", "error", err)
		return nil
	}

	return nil
}

// Stop halts media playback in a voice chat and clears the chat's cache.
func (c *TelegramCalls) Stop(chatId int64) error {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return err
	}

	cache.ChatCache.ClearChat(chatId)
	err = call.stopCall(context.Background(), chatId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}

		slog.Info("[Stop] Failed to stop the call", "error", err, "index", index)
		return fmt.Errorf("failed to stop call: %w", err)
	}
	return nil
}

// Pause temporarily stops media playback in a voice chat.
// It returns true if the operation was successful, and an error otherwise.
func (c *TelegramCalls) Pause(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.Pause(chatId)
	if err != nil {
		slog.Warn("[Pause] Failed to pause the call", "error", err, "index", index)
		return res, fmt.Errorf("failed to pause: %w", err)
	}
	return res, err
}

// Resume continues a paused media playback in a voice chat.
func (c *TelegramCalls) Resume(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.Resume(chatId)
	if err != nil {
		logger.Warn("Failed to resume the call", "error", err, "index", index)
		return res, fmt.Errorf("failed to resume: %w", err)
	}

	return res, err
}

// Mute silences the media playback in a voice chat.
func (c *TelegramCalls) Mute(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.Mute(chatId)
	if err != nil {
		logger.Warn("Failed to mute the call", "error", err, "index", index)
		return res, fmt.Errorf("failed to mute: %w", err)
	}

	return res, err
}

// Unmute restores the audio of a muted media playback in a voice chat.
func (c *TelegramCalls) Unmute(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.UnMute(chatId)
	if err != nil {
		logger.Warn("Failed to unmute the call", "error", err, "index", index)
		return res, fmt.Errorf("failed to unmute: %w", err)
	}

	return res, err
}

// PlayedTime retrieves the elapsed time of the current playback in a voice chat.
func (c *TelegramCalls) PlayedTime(chatId int64) (uint64, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return 0, err
	}

	_time, err := call.binding.Time(chatId, 0)
	if err != nil {
		logger.Warn("Failed to get played time", "error", err, "index", index)
		return 0, fmt.Errorf("failed to get played time: %w", err)
	}

	return _time, nil
}

// SeekStream jumps to a specific time in the current media stream.
func (c *TelegramCalls) SeekStream(chatID int64, filePath string, toSeek, duration int, isVideo bool) error {
	if toSeek < 0 || duration <= 0 {
		return errors.New("invalid seek position or duration. The position must be positive and the duration must be greater than 0")
	}

	isURL := urlRegex.MatchString(filePath)
	_, err := os.Stat(filePath)
	isFile := err == nil

	var ffmpegParams string
	if isURL || !isFile {
		ffmpegParams = fmt.Sprintf("-ss %d -i %s -to %d", toSeek, filePath, duration)
	} else {
		ffmpegParams = fmt.Sprintf("-ss %d -to %d", toSeek, duration)
	}

	return c.PlayMedia(chatID, filePath, isVideo, ffmpegParams)
}

// ChangeSpeed modifies the playback speed of the current stream.
func (c *TelegramCalls) ChangeSpeed(chatID int64, speed float64) error {
	if speed < 0.5 || speed > 4.0 {
		return errors.New("invalid speed. Value must be between 0.5 and 4.0")
	}

	playingSong := cache.ChatCache.GetPlayingTrack(chatID)
	if playingSong == nil {
		return errors.New("the bot isn't streaming in the video chat")
	}

	videoPTS := 1 / speed

	var audioFilterBuilder strings.Builder
	remaining := speed
	for remaining > 2.0 {
		audioFilterBuilder.WriteString("atempo=2.0,")
		remaining /= 2.0
	}
	for remaining < 0.5 {
		audioFilterBuilder.WriteString("atempo=0.5,")
		remaining /= 0.5
	}
	audioFilterBuilder.WriteString(fmt.Sprintf("atempo=%f", remaining))
	audioFilter := audioFilterBuilder.String()

	ffmpegFilters := fmt.Sprintf("-filter:v setpts=%f*PTS -filter:a %s", videoPTS, audioFilter)
	return c.PlayMedia(chatID, playingSong.FilePath, playingSong.IsVideo, ffmpegFilters)
}

// RegisterHandlers sets up the event handlers for the voice call client.
func (c *TelegramCalls) RegisterHandlers(client *td.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.bot = client

	c.startAutoLeave(context.Background())

	for _, call := range c.assistants {
		call.OnStreamEnd(func(chatID int64, streamType ntgcalls.StreamType, device ntgcalls.StreamDevice) {
			if streamType == ntgcalls.VideoStream {
				return
			}

			if err := c.PlayNext(chatID); err != nil {
				call.App.Logger.Warnf("[OnStreamEnd] Failed to play the song: %v", err)
			}
		})

		_, err := call.App.SendMessage(client.Me.Usernames.EditableUsername, "/start")
		if err != nil {
			call.App.Logger.Warnf("failed to start bot: %v", err)
		}

		_, err = call.App.SendMessage(config.LoggerId, "Userbot started.")
		if err != nil {
			call.App.Logger.Warnf("Failed to send message: %v", err)
		}
	}
}
