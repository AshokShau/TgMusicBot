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
	"ashokshau/tgmusic/src/utils"
	"ashokshau/tgmusic/src/vc/ntgcalls"
	"ashokshau/tgmusic/src/vc/sessions"
	"ashokshau/tgmusic/src/vc/ubot"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"regexp"
	"strings"
	"time"

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/core/dl"

	td "github.com/AshokShau/gotdbot"
	tg "github.com/amarnathcjd/gogram/telegram"
)

const DefaultStreamURL = "https://t.me/FallenSongs/1295"

// getClientIndex selects an assistant client index for a given chat. It prioritizes existing assignments from the database.
func (c *TelegramCalls) getClientIndex(chatID int64, excludeIndices []int) (int, error) {
	c.mu.RLock()
	totalClients := len(c.availableClients)
	c.mu.RUnlock()

	if totalClients == 0 {
		return 0, fmt.Errorf("no clients are available")
	}

	var availableIndices []int
	for i := 1; i <= totalClients; i++ {
		excluded := false
		for _, ex := range excludeIndices {
			if i == ex {
				excluded = true
				break
			}
		}
		if !excluded {
			availableIndices = append(availableIndices, i)
		}
	}

	if len(availableIndices) == 0 {
		// Fallback if all are excluded
		for i := 1; i <= totalClients; i++ {
			availableIndices = append(availableIndices, i)
		}
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(availableIndices))))
	if err != nil {
		slog.Info("[TelegramCalls] Could not generate a random number", "error", err)
		return availableIndices[0], nil
	}
	newClientIndex := availableIndices[n.Int64()]

	ctx, cancel := db.Ctx()
	defer cancel()

	assignedIndex, err := db.Instance.AssignAssistant(ctx, chatID, newClientIndex)
	if err != nil {
		slog.Info("[TelegramCalls] DB.AssignAssistant error", "error", err)
	}

	if assignedIndex != 0 {
		isAvailable := false
		for _, idx := range availableIndices {
			if idx == assignedIndex {
				isAvailable = true
				break
			}
		}

		if isAvailable {
			return assignedIndex, nil
		}

		slog.Info("[TelegramCalls] Assigned assistant is unavailable or excluded. Overwriting with .", "arg1", assignedIndex, "arg2", newClientIndex)
		if err = db.Instance.SetAssistant(ctx, chatID, newClientIndex); err != nil {
			slog.Info("[TelegramCalls] DB.SetAssistant error", "error", err)
		}
		return newClientIndex, nil
	}

	if err = db.Instance.SetAssistant(ctx, chatID, newClientIndex); err != nil {
		slog.Info("[TelegramCalls] DB.SetAssistant error", "error", err)
	}

	slog.Info("[TelegramCalls] An assistant has been set for chat  ->", "id", chatID, "arg2", newClientIndex)
	return newClientIndex, nil
}

// GetGroupAssistant retrieves the ubot.Context for a given chat, which is used to interact with the voice call.
func (c *TelegramCalls) GetGroupAssistant(chatID int64, excludeIndices ...int) (*ubot.Context, error) {
	clientIndex, err := c.getClientIndex(chatID, excludeIndices)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	clientName := fmt.Sprintf("client%d", clientIndex-1)
	call, ok := c.uBContext[clientName]
	if !ok {
		return nil, fmt.Errorf("no ntgcalls instance was found for %s (index %d)", clientName, clientIndex)
	}
	return call, nil
}

// StartClient initializes a new userbot client and adds it to the pool of available assistants.
// It authenticates with Telegram using the provided API ID, API hash, and session string.
// The session type is determined by the configuration (pyrogram, telethon, or gogram).
func (c *TelegramCalls) StartClient(apiID int32, apiHash, stringSession string) (*ubot.Context, error) {
	c.mu.Lock()
	clientName := fmt.Sprintf("client%d", c.clientCounter)
	c.clientCounter++
	c.mu.Unlock()

	var sess *tg.Session
	var err error

	clientConfig := tg.ClientConfig{
		AppID:         apiID,
		AppHash:       apiHash,
		MemorySession: true,
		SessionName:   clientName,
		FloodHandler:  handleFlood,
		LogLevel:      tg.InfoLevel,
	}

	switch config.Conf.SessionType {
	case "telethon":
		sess, err = sessions.DecodeTelethonSessionString(stringSession)
		if err != nil {
			return nil, fmt.Errorf("failed to decode telethon session string for %s: %w", clientName, err)
		}
		clientConfig.StringSession = sess.Encode()
	case "pyrogram":
		sess, err = sessions.DecodePyrogramSessionString(stringSession)
		if err != nil {
			return nil, fmt.Errorf("failed to decode pyrogram session string for %s: %w", clientName, err)
		}
		clientConfig.StringSession = sess.Encode()
	case "gogram":
		clientConfig.StringSession = stringSession
	default:
		return nil, fmt.Errorf("unsupported session type: %s", config.Conf.SessionType)
	}

	mtProto, err := tg.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create the MTProto client: %w", err)
	}

	if err := mtProto.Start(); err != nil {
		return nil, fmt.Errorf("failed to start the client: %w", err)
	}

	if mtProto.Me().Bot {
		_ = mtProto.Stop()
		return nil, fmt.Errorf("the client %s is a bot", clientName)
	}

	call, err := ubot.NewInstance(mtProto)
	if err != nil {
		_ = mtProto.Stop()
		return nil, fmt.Errorf("failed to create the ubot instance: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.uBContext[clientName] = call
	c.clients[clientName] = mtProto
	c.availableClients = append(c.availableClients, clientName)

	mtProto.Logger.Info("[TelegramCalls] client %s has started successfully.", clientName)
	return call, nil
}

// StopAllClients gracefully stops all active userbot clients and their associated voice calls.
func (c *TelegramCalls) StopAllClients() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, call := range c.uBContext {
		call.Close()
	}

	for name, client := range c.clients {
		slog.Info("[TelegramCalls] Stopping the client", "arg1", name)
		_ = client.Stop()
	}
}

// PlayMedia plays media in a voice chat using ffmpeg. It downloads the file if necessary
// and updates the cache and logger status.
func (c *TelegramCalls) PlayMedia(chatID int64, filePath string, video bool, ffmpegParameters string) error {
	call, err := c.GetGroupAssistant(chatID)
	if err != nil {
		return err
	}
	ctx, cancel := db.Ctx()
	defer cancel()

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

	slog.Debug("Playing media in chat", "id", chatID, "path", filePath)

	mediaDesc := getMediaDescription(filePath, video, ffmpegParameters)
	if err = call.Play(chatID, mediaDesc); err != nil {
		logger.Error("Failed to play the media", "error", err)
		cache.ChatCache.ClearChat(chatID)
		return fmt.Errorf("playback failed: %w", err)
	}

	if db.Instance.GetLoggerStatus(ctx, c.bot.Me().Id) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	dlPath, err := dl.DownloadSong(ctx, song, c.bot)
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
		_, err := reply.EditText(c.bot, err.Error(), nil)
		return err
	}

	if song.Duration == 0 {
		song.Duration = utils.GetMediaDuration(song.FilePath)
	}

	text := fmt.Sprintf(
		"<b>Now Playing:</b>\n\n<b>Track:</b> <a href='%s'>%s</a>\n<b>Duration:</b> %s\n<b>By:</b> %s",
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
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return err
	}
	cache.ChatCache.ClearChat(chatId)
	err = call.Stop(chatId)
	if err != nil {
		slog.Info("[Stop] Failed to stop the call", "error", err)
		return err
	}
	return nil
}

// Pause temporarily stops media playback in a voice chat.
// It returns true if the operation was successful, and an error otherwise.
func (c *TelegramCalls) Pause(chatId int64) (bool, error) {
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.Pause(chatId)
	if err != nil {
		slog.Warn("[Pause] Failed to pause the call", "error", err)
	}
	return res, err
}

// Resume continues a paused media playback in a voice chat.
// It returns true if the operation was successful, and an error otherwise.
func (c *TelegramCalls) Resume(chatId int64) (bool, error) {
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}
	res, err := call.Resume(chatId)
	if err != nil {
		slog.Warn("[Resume] Failed to resume the call", "error", err)
	}
	return res, err
}

// Mute silences the media playback in a voice chat.
// It returns true if the operation was successful, and an error otherwise.
func (c *TelegramCalls) Mute(chatId int64) (bool, error) {
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}
	res, err := call.Mute(chatId)
	if err != nil {
		slog.Warn("[Mute] Failed to mute the call", "error", err)
	}
	return res, err
}

// Unmute restores the audio of a muted media playback in a voice chat.
// It returns true if the operation was successful, and an error otherwise.
func (c *TelegramCalls) Unmute(chatId int64) (bool, error) {
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}
	res, err := call.Unmute(chatId)
	if err != nil {
		slog.Warn("[Unmute] Failed to unmute the call", "error", err)
	}
	return res, err
}

// PlayedTime retrieves the elapsed time of the current playback in a voice chat.
// It returns the elapsed time in seconds and an error if any.
func (c *TelegramCalls) PlayedTime(chatId int64) (uint64, error) {
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return 0, err
	}

	// TODO: Pass the streamMode.
	return call.Time(chatId, 0)
}

var urlRegex = regexp.MustCompile(`^https?://`)

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
		return errors.New("🔇 Nothing is playing")
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

		//_, _ = call.App.UpdatesGetState()
		call.OnStreamEnd(func(chatID int64, streamType ntgcalls.StreamType, device ntgcalls.StreamDevice) {
			call.App.Logger.Warnf("[TelegramCalls] The stream has ended in chat %d (type=%v, device=%v)", chatID, streamType, device)
			if streamType == ntgcalls.VideoStream {
				call.App.Logger.Warnf("Ignoring video stream end for chat %d", chatID)
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

		//call.OnFrame(func(chatId int64, mode ntgcalls.StreamMode, device ntgcalls.StreamDevice, frames []ntgcalls.Frame) {
		//	call.App.Logger.Infof("Received frames for chatId: %d, mode: %v, device: %v", chatId, mode, device)
		//})

		_, _ = call.App.SendMessage(client.Me().Usernames.EditableUsername, "/start")
		_, err := call.App.SendMessage(config.Conf.LoggerId, "Userbot started.")
		if err != nil {
			call.App.Logger.Infof("[TelegramCalls - SendMessage] Failed to send message: %v", err)
		}
	}
}
