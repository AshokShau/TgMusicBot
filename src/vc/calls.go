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
	"ashokshau/tgmusic/src/core/dl"
	"ashokshau/tgmusic/src/utils"
	"ashokshau/tgmusic/src/vc/ntgcalls"
	"ashokshau/tgmusic/src/vc/ubot"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

const DefaultStreamURL = "https://t.me/FallenSongs/1295"

// getClientIndex selects an assistant client index (0-based) for a given chat.
// It prioritizes existing assignments from the database.
func (c *TelegramCalls) getClientIndex(chatID int64) (int, error) {
	c.mu.RLock()
	totalClients := len(c.uBContext)
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

// GetGroupAssistant retrieves the ubot.Context and its index for a given chat.
func (c *TelegramCalls) GetGroupAssistant(chatID int64) (*ubot.Context, int, error) {
	clientIndex, err := c.getClientIndex(chatID)
	if err != nil {
		return nil, -1, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	call, ok := c.uBContext[clientIndex]
	if !ok {
		return nil, -1, fmt.Errorf("no ntgcalls instance was found for client index %d", clientIndex)
	}
	return call, clientIndex, nil
}

// PlayMedia plays media in a voice chat.
func (c *TelegramCalls) PlayMedia(chatID int64, filePath string, video bool, ffmpegParameters string) error {
	c.mu.RLock()
	totalClients := len(c.uBContext)
	c.mu.RUnlock()

	if totalClients == 0 {
		return fmt.Errorf("no clients are available")
	}

	var lastErr error
	for i := 0; i < totalClients; i++ {
		err := c.play(chatID, filePath, video, ffmpegParameters)
		if err == nil {
			return nil
		}

		errMsg := err.Error()
		if strings.Contains(errMsg, "FROZEN_METHOD_INVALID") ||
			strings.Contains(errMsg, "CHANNELS_TOO_MUCH") ||
			strings.Contains(errMsg, "FLOOD_WAIT_X") {
			_ = db.Instance.RemoveAssistant(chatID)
			lastErr = err
			continue
		}

		return err
	}

	return lastErr
}

func (c *TelegramCalls) play(chatID int64, filePath string, video bool, ffmpegParameters string) error {
	call, index, err := c.GetGroupAssistant(chatID)
	if err != nil {
		return err
	}

	if chatID < 0 {
		var joinErr error
		call, joinErr = c.JoinAssistant(chatID)
		if joinErr != nil {
			cache.ChatCache.ClearChat(chatID)
			return joinErr
		}
	} else {
		_, _ = call.App.ResolvePeer(chatID)
	}

	logger.Debug("Playing media in chat", "id", chatID, "path", filePath, "index", index)
	mediaDesc := getMediaDescription(filePath, video, ffmpegParameters)
	if err = call.Play(chatID, mediaDesc); err != nil {
		cache.ChatCache.ClearChat(chatID)
		if strings.Contains(err.Error(), "is closed") {
			return errors.New("<b>No active video chat found.</b>\n\nPlease start one and <b>try again</b>")
		}

		logger.Error("Failed to play the media", "error", err, "index", index)
		return fmt.Errorf("client%d: playback failed: %w", index, err)
	}

	if db.Instance.GetLoggerStatus() {
		go sendLogger(c.bot, chatID, cache.ChatCache.GetPlayingTrack(chatID))
	}

	return nil
}

// downloadAndPrepareSong handles the download and preparation of a song for playback.
// It returns an error if the download or preparation fails.
func (c *TelegramCalls) downloadAndPrepareSong(song *utils.CachedTrack, reply *td.Message) error {
	if song.FilePath != "" {
		return nil
	}

	dlPath, err := dl.DownloadCachedTrack(song, c.bot)
	if err != nil {
		_, _ = reply.EditText(c.bot, "⚠️ Download failed. Skipping track...", nil)
		return err
	}

	song.FilePath = dlPath
	if song.FilePath == "" {
		_, _ = reply.EditText(c.bot, "⚠️ Download failed. Skipping track...", nil)
		return errors.New("download failed due to an empty file path")
	}

	return nil
}

// PlayNext plays the next song in the queue, handles looping, and notifies the chat when the queue is finished.
func (c *TelegramCalls) PlayNext(chatID int64) error {
	loop := cache.ChatCache.GetLoopCount(chatID)
	if loop > 0 {
		cache.ChatCache.SetLoopCount(chatID, loop-1)
		if currentsSong := cache.ChatCache.GetPlayingTrack(chatID); currentsSong != nil {
			return c.playSong(chatID, currentsSong)
		}
	}

	if nextSong := cache.ChatCache.GetUpcomingTrack(chatID); nextSong != nil {
		cache.ChatCache.RemoveCurrentSong(chatID)
		return c.playSong(chatID, nextSong)
	}

	cache.ChatCache.RemoveCurrentSong(chatID)
	return c.handleNoSong(chatID)
}

// handleNoSong manages the situation where there are no more songs in the queue by stopping the playback
// and sending a notification to the chat.
func (c *TelegramCalls) handleNoSong(chatID int64) error {
	_ = c.Stop(chatID)
	_, _ = c.bot.SendTextMessage(chatID, "🎵 Queue finished. Add more songs with /play.", nil)
	return nil
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
		_, err := reply.EditText(c.bot, err.Error(), &td.EditTextMessageOpts{ParseMode: "HTML", DisableWebPagePreview: true})
		return err
	}

	if song.Duration == 0 {
		song.Duration = utils.GetMediaDuration(song.FilePath)
	}

	text := fmt.Sprintf(
		"<u><b>| Started streaming</b></u>\n\n<b>Title:</b> <a href='%s'>%s</a>\n\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
		song.URL,
		song.Name,
		utils.SecToMin(song.Duration),
		song.User,
	)

	_, err = reply.EditText(c.bot, text, &td.EditTextMessageOpts{
		ReplyMarkup:           core.ControlButtons("play"),
		ParseMode:             "HTMl",
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
	err = call.Stop(chatId)
	if err != nil {
		slog.Info("[Stop] Failed to stop the call", "error", err, "index", index)
		return fmt.Errorf("failed to stop call (client %d): %w", index, err)
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

	res, err := call.Pause(chatId)
	if err != nil {
		slog.Warn("[Pause] Failed to pause the call", "error", err, "index", index)
		return res, fmt.Errorf("failed to pause (client %d): %w", index, err)
	}
	return res, err
}

// Resume continues a paused media playback in a voice chat.
func (c *TelegramCalls) Resume(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.Resume(chatId)
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

	res, err := call.Mute(chatId)
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

	res, err := call.Unmute(chatId)
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

	_time, err := call.Time(chatId, 0)
	if err != nil {
		logger.Warn("Failed to get played time", "error", err, "index", index)
		return 0, fmt.Errorf("failed to get played time: %w", err)
	}

	return _time, nil
}

// CpuUsage Get an estimate of the CPU usage of the current process.
func (c *TelegramCalls) CpuUsage(chatId int64) (float64, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return 0, err
	}

	usage, err := call.CpuUsage()
	if err != nil {
		logger.Warn("Failed to get CPU usage", "error", err, "index", index)
		return 0, fmt.Errorf("failed to get cpu usage: %w", err)
	}

	return usage, nil
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

	for _, call := range c.uBContext {
		call.OnStreamEnd(func(chatID int64, streamType ntgcalls.StreamType, device ntgcalls.StreamDevice) {
			if streamType == ntgcalls.VideoStream {
				return
			}

			if err := c.PlayNext(chatID); err != nil {
				call.App.Logger.Warnf("[OnStreamEnd] Failed to play the song: %v", err)
			}
		})

		call.OnIncomingCall(func(ub *ubot.Context, chatID int64) {
			_, _ = ub.App.SendMessage(chatID, "Incoming call detected. Playing music...")
			msg, err := utils.GetMessage(c.bot, DefaultStreamURL)
			if err != nil {
				call.App.Logger.Warnf("[OnIncomingCall] Failed to get the message: %v", err)
				return
			}

			file, err := msg.Download(c.bot, 1, 0, 0, true)
			if err != nil {
				call.App.Logger.Warnf("[OnIncomingCall] Failed to download the message: %v", err)
				return
			}

			err = c.PlayMedia(chatID, file.Local.Path, false, "")
			if err != nil {
				call.App.Logger.Warnf("[OnIncomingCall] Failed to play the media: %v", err)
				return
			}

			return
		})

		_, err := call.App.SendMessage(client.Me.Usernames.EditableUsername, "/start")
		if err != nil {
			call.App.Logger.Warnf("failed to start bot: %v", err)
		}

		_, err = call.App.SendMessage(config.Conf.LoggerId, "Userbot started.")
		if err != nil {
			call.App.Logger.Warnf("Failed to send message: %v", err)
		}
	}
}
