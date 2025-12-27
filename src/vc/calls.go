/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
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
	"log"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/core/dl"

	tg "github.com/amarnathcjd/gogram/telegram"
)

const DefaultStreamURL = "https://t.me/FallenSongs/1295"

// getClientName selects an assistant client for a given chat. It prioritizes existing assignments from the database.
func (c *TelegramCalls) getClientName(chatID int64) (string, error) {
	c.mu.RLock()
	if len(c.availableClients) == 0 {
		c.mu.RUnlock()
		return "", fmt.Errorf("no clients are available")
	}
	availableClients := make([]string, len(c.availableClients))
	copy(availableClients, c.availableClients)
	c.mu.RUnlock()

	ctx, cancel := db.Ctx()
	defer cancel()

	assistant, err := db.Instance.GetAssistant(ctx, chatID)
	if err != nil {
		c.bot.Log.Info("[TelegramCalls] DB.GetAssistant error: %v", err)
	}

	if assistant != "" {
		for _, name := range availableClients {
			if name == assistant {
				return name, nil
			}
		}
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(availableClients))))
	if err != nil {
		log.Printf("[TelegramCalls] Could not generate a random number: %v", err)
		return availableClients[0], nil
	}
	newClient := availableClients[n.Int64()]

	if err = db.Instance.SetAssistant(ctx, chatID, newClient); err != nil {
		c.bot.Log.Info("[TelegramCalls] DB.SetAssistant error: %v", err)
	}

	c.bot.Log.Info("[TelegramCalls] An assistant has been set for chat %d -> %s", chatID, newClient)
	return newClient, nil
}

// GetGroupAssistant retrieves the ubot.Context for a given chat, which is used to interact with the voice call.
func (c *TelegramCalls) GetGroupAssistant(chatID int64) (*ubot.Context, error) {
	clientName, err := c.getClientName(chatID)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	call, ok := c.uBContext[clientName]
	if !ok {
		return nil, fmt.Errorf("no ntgcalls instance was found for %s", clientName)
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
		LogLevel:      tg.TraceLevel,
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
		c.bot.Log.Info("[TelegramCalls] Stopping the client: %s", name)
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
		if err := c.joinAssistant(chatID, call.App.Me().ID); err != nil {
			cache.ChatCache.ClearChat(chatID)
			return err
		}
	} else {
		_, _ = call.App.ResolvePeer(chatID)
	}

	c.bot.Log.Info("Playing media in chat %d: %s", chatID, filePath)
	mediaDesc := getMediaDescription(filePath, video, ffmpegParameters)
	if err := call.Play(chatID, mediaDesc); err != nil {
		logger.Error("Failed to play the media: %v", err)
		cache.ChatCache.ClearChat(chatID)
		return fmt.Errorf("playback failed: %w", err)
	}

	if db.Instance.GetLoggerStatus(ctx, c.bot.Me().ID) {
		go sendLogger(c.bot, chatID, cache.ChatCache.GetPlayingTrack(chatID))
	}

	return nil
}

// downloadAndPrepareSong handles the download and preparation of a song for playback.
// It returns an error if the download or preparation fails.
func (c *TelegramCalls) downloadAndPrepareSong(song *utils.CachedTrack, reply *tg.NewMessage) error {
	if song.FilePath != "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	dlPath, err := dl.DownloadSong(ctx, song, c.bot)
	if err != nil {
		_, _ = reply.Edit("⚠️ Download failed. Skipping track...")
		return err
	}

	song.FilePath = dlPath
	if song.FilePath == "" {
		_, _ = reply.Edit("⚠️ Download failed. Skipping track...")
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
	_, _ = c.bot.SendMessage(chatID, "🎵 Queue finished. Add more songs with /play.")
	return nil
}

// playSong downloads and plays a single song. It sends a message to the chat to indicate the download status
// and updates it with the song's information once playback begins.
func (c *TelegramCalls) playSong(chatID int64, song *utils.CachedTrack) error {
	reply, err := c.bot.SendMessage(chatID, fmt.Sprintf("Downloading %s...", song.Name))
	if err != nil {
		c.bot.Log.Info("[playSong] Failed to send message: %v", err)
		return err
	}

	if err := c.downloadAndPrepareSong(song, reply); err != nil {
		return c.PlayNext(chatID)
	}

	if err := c.PlayMedia(chatID, song.FilePath, song.IsVideo, ""); err != nil {
		_, err := reply.Edit(err.Error())
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

	_, err = reply.Edit(text, &tg.SendOptions{
		ReplyMarkup: core.ControlButtons("play"),
	})
	if err != nil {
		c.bot.Log.Warn("[playSong] Failed to edit message: %v", err)
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
		c.bot.Log.Error("[Stop] Failed to stop the call: %v", err)
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
		c.bot.Log.Error("[Pause] Failed to pause the call: %v", err)
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
		c.bot.Log.Error("[Resume] Failed to resume the call: %v", err)
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
		c.bot.Log.Error("[Mute] Failed to mute the call: %v", err)
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
		c.bot.Log.Error("[Unmute] Failed to unmute the call: %v", err)
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
func (c *TelegramCalls) RegisterHandlers(client *tg.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.bot = client
	logger = client.Log

	for _, call := range c.uBContext {

		//_, _ = call.App.UpdatesGetState()
		call.OnStreamEnd(func(chatID int64, streamType ntgcalls.StreamType, device ntgcalls.StreamDevice) {
			client.Log.Info("[TelegramCalls] The stream has ended in chat %d (type=%v, device=%v)", chatID, streamType, device)
			if streamType == ntgcalls.VideoStream {
				client.Log.Info("Ignoring video stream end for chat %d", chatID)
				return
			}

			if err := c.PlayNext(chatID); err != nil {
				client.Log.Error("[OnStreamEnd] Failed to play the song: %v", err)
			}
		})

		call.OnIncomingCall(func(ub *ubot.Context, chatID int64) {
			_, _ = ub.App.SendMessage(chatID, "Incoming call detected. Playing music...")
			msg, err := utils.GetMessage(c.bot, DefaultStreamURL)
			if err != nil {
				c.bot.Log.Info("[OnIncomingCall] Failed to get the message: %v", err)
				return
			}

			dCtx, dCancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer dCancel()
			filePath, err := msg.Download(&tg.DownloadOptions{FileName: filepath.Join(config.Conf.DownloadsDir, msg.File.Name), Ctx: dCtx})
			if err != nil {
				c.bot.Log.Info("[OnIncomingCall] Failed to download the message: %v", err)
				return
			}

			err = c.PlayMedia(chatID, filePath, false, "")
			if err != nil {
				c.bot.Log.Info("[OnIncomingCall] Failed to play the media: %v", err)
				return
			}

			return
		})

		call.OnFrame(func(chatId int64, mode ntgcalls.StreamMode, device ntgcalls.StreamDevice, frames []ntgcalls.Frame) {
			c.bot.Log.Debug("Received frames for chatId: %d, mode: %v, device: %v", chatId, mode, device)
		})

		_, _ = call.App.SendMessage(client.Me().Username, "/start")
		_, err := call.App.SendMessage(config.Conf.LoggerId, "Userbot started.")
		if err != nil {
			c.bot.Log.Info("[TelegramCalls - SendMessage] Failed to send message: %v", err)
		}
	}
}
