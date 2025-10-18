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
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"tgmusic/pkg/config"
	"tgmusic/pkg/core"
	"tgmusic/pkg/core/cache"
	"tgmusic/pkg/core/db"
	"tgmusic/pkg/core/dl"
	"tgmusic/pkg/vc/ntgcalls"
	"tgmusic/pkg/vc/ubot"
	"time"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// addBot registers the bot's client, enabling it to send messages and perform other actions.
func (c *TelegramCalls) addBot(bot *tg.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bot = bot
	gologging.Info("The bot client has been added.")
}

// getClientName selects an assistant client for a given chat. It prioritizes existing assignments from the database.
// If no assignment exists, it randomly selects an available client and saves the assignment for future use.
//
// TODO: Implement a more sophisticated client selection strategy, such as consistent hashing or load-based balancing,
// to ensure a more even distribution of chats among assistants.
func (c *TelegramCalls) getClientName(chatID int64) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.availableClients) == 0 {
		return "", fmt.Errorf("no clients are available")
	}
	ctx, cancel := db.Ctx()
	defer cancel()

	assistant, err := db.Instance.GetAssistant(ctx, chatID)
	if err != nil {
		gologging.InfoF("[TelegramCalls] DB.GetAssistant error: %v", err)
	}

	if assistant != "" {
		for _, name := range c.availableClients {
			if name == assistant {
				return name, nil
			}
		}
	}

	newClient := c.availableClients[rand.Intn(len(c.availableClients))]
	if err := db.Instance.SetAssistant(ctx, chatID, newClient); err != nil {
		gologging.InfoF("[TelegramCalls] DB.SetAssistant error: %v", err)
	}

	gologging.InfoF("[TelegramCalls] An assistant has been set for chat %d -> %s", chatID, newClient)
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
func (c *TelegramCalls) StartClient(apiID int32, apiHash, stringSession string) (*ubot.Context, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	clientName := fmt.Sprintf("client%d", c.clientCounter)
	sess, err := decodePyrogramSessionString(stringSession)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while decoding the session string for %s: %v", clientName, err)
	}

	mtProto, err := tg.NewClient(tg.ClientConfig{
		AppID:         apiID,
		AppHash:       apiHash,
		StringSession: sess.Encode(),
		MemorySession: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create the MTProto client: %w", err)
	}

	if err := mtProto.Start(); err != nil {
		return nil, fmt.Errorf("failed to start the client: %w", err)
	}

	if mtProto.Me().Bot {
		return nil, fmt.Errorf("the client %s is a bot", clientName)
	}

	call, err := ubot.NewInstance(mtProto)
	if err != nil {
		return nil, fmt.Errorf("failed to create the ubot instance: %w", err)
	}

	c.uBContext[clientName] = call
	c.clients[clientName] = mtProto
	c.availableClients = append(c.availableClients, clientName)
	c.clientCounter++

	gologging.InfoF("[TelegramCalls] Client %s has started successfully.", clientName)
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
		gologging.InfoF("[TelegramCalls] Stopping the client: %s", name)
		_ = client.Stop()
	}
}

// PlayMedia starts playing a media file in a voice chat. It handles joining the assistant to the chat if necessary
// and sends a log message if logging is enabled.
func (c *TelegramCalls) PlayMedia(chatID int64, filePath string, video bool, ffmpegParameters string) error {
	call, err := c.GetGroupAssistant(chatID)
	if err != nil {
		return err
	}
	ctx, cancel := db.Ctx()
	defer cancel()

	if chatID < 0 {
		if err := c.joinAssistant(chatID, call.App.Me().ID); err != nil {
			cache.ChatCache.ClearChat(chatID, true)
			return err
		}
	} else {
		_, _ = call.App.ResolvePeer(chatID)
	}

	gologging.InfoF("Playing media in chat %d: %s", chatID, filePath)
	mediaDesc := getMediaDescription(filePath, video, ffmpegParameters)
	if err := call.Play(chatID, mediaDesc); err != nil {
		gologging.ErrorF("Failed to play the media: %v", err)
		cache.ChatCache.ClearChat(chatID, true)
		return fmt.Errorf("playback failed: %w", err)
	}

	if db.Instance.GetLoggerStatus(ctx, c.bot.Me().ID) {
		go sendLogger(c.bot, chatID, cache.ChatCache.GetPlayingTrack(chatID))
	}

	return nil
}

// downloadAndPrepareSong handles the download and preparation of a song for playback.
// It returns an error if the download or preparation fails.
func (c *TelegramCalls) downloadAndPrepareSong(song *cache.CachedTrack, reply *tg.NewMessage) error {
	if song.FilePath != "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	dlPath, trackInfo, err := DownloadSong(ctx, song, c.bot)
	if err != nil {
		_, _ = reply.Edit(fmt.Sprintf("⚠️ Failed to download the song: (%v)\nSkipping to the next track...", err))
		return err
	}

	song.FilePath = dlPath
	if trackInfo != nil && trackInfo.Duration > 0 {
		song.Duration = trackInfo.Duration
	}

	if song.FilePath == "" {
		_, _ = reply.Edit("⚠️ Failed to download the song.\nSkipping to the next track...")
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
		cache.ChatCache.RemoveCurrentSong(chatID, true)
		return c.playSong(chatID, nextSong)
	}

	cache.ChatCache.RemoveCurrentSong(chatID, true)
	return c.handleNoSong(chatID)
}

// handleNoSong manages the situation where there are no more songs in the queue by stopping the playback
// and sending a notification to the chat.
func (c *TelegramCalls) handleNoSong(chatId int64) error {
	_ = c.Stop(chatId)
	_, _ = c.bot.SendMessage(chatId, "🎵 The queue has finished. Use /play to add more songs!")
	return nil
}

// playSong downloads and plays a single song. It sends a message to the chat to indicate the download status
// and updates it with the song's information once playback begins.
func (c *TelegramCalls) playSong(chatID int64, song *cache.CachedTrack) error {
	reply, err := c.bot.SendMessage(chatID, fmt.Sprintf("Downloading %s...", song.Name))
	if err != nil {
		gologging.InfoF("[playSong] Failed to send message: %v", err)
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
		song.Duration = cache.GetFileDuration(song.FilePath)
	}
	text := fmt.Sprintf(
		"<b>Now Playing:</b>\n\n‣ <b>Title:</b> <a href='%s'>%s</a>\n‣ <b>Duration:</b> %s\n‣ <b>Requested by:</b> %s",
		song.URL,
		song.Name,
		cache.SecToMin(song.Duration),
		song.User,
	)

	_, err = reply.Edit(text, tg.SendOptions{ReplyMarkup: core.ControlButtons("play")})
	if err != nil {
		gologging.InfoF("[playSong] Failed to edit message: %v", err)
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
	cache.ChatCache.ClearChat(chatId, true)
	err = call.Stop(chatId)
	if err != nil {
		gologging.InfoF("[Stop] Failed to stop the call: %v", err)
		// For now, we will ignore the error.
		return nil
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

	return call.Pause(chatId)
}

// Resume continues a paused media playback in a voice chat.
// It returns true if the operation was successful, and an error otherwise.
func (c *TelegramCalls) Resume(chatId int64) (bool, error) {
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}
	return call.Resume(chatId)
}

// Mute silences the media playback in a voice chat.
// It returns true if the operation was successful, and an error otherwise.
func (c *TelegramCalls) Mute(chatId int64) (bool, error) {
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}
	return call.Mute(chatId)
}

// Unmute restores the audio of a muted media playback in a voice chat.
// It returns true if the operation was successful, and an error otherwise.
func (c *TelegramCalls) Unmute(chatId int64) (bool, error) {
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}
	return call.Unmute(chatId)
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
func (c *TelegramCalls) SeekStream(chatId int64, filePath string, toSeek, duration int, isVideo bool) error {
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

	return c.PlayMedia(chatId, filePath, isVideo, ffmpegParams)
}

// ChangeSpeed modifies the playback speed of the current stream.
func (c *TelegramCalls) ChangeSpeed(chatId int64, speed float64) error {
	if speed < 0.5 || speed > 4.0 {
		return errors.New("invalid speed: the value must be between 0.5 and 4.0")
	}

	playingSong := cache.ChatCache.GetPlayingTrack(chatId)
	if playingSong == nil {
		return errors.New("no song is currently playing")
	}

	videoPTS := 1 / speed

	audioFilters := make([]string, 0)
	remaining := speed
	for remaining > 2.0 {
		audioFilters = append(audioFilters, "atempo=2.0")
		remaining /= 2.0
	}
	for remaining < 0.5 {
		audioFilters = append(audioFilters, "atempo=0.5")
		remaining /= 0.5
	}
	audioFilters = append(audioFilters, fmt.Sprintf("atempo=%f", remaining))
	audioFilter := strings.Join(audioFilters, ",")

	ffmpegFilters := fmt.Sprintf("-filter:v setpts=%f*PTS -filter:a %s", videoPTS, audioFilter)

	return c.PlayMedia(chatId, playingSong.FilePath, playingSong.IsVideo, ffmpegFilters)
}

// RegisterHandlers sets up the event handlers for the voice call client.
func (c *TelegramCalls) RegisterHandlers(client *tg.Client) {
	c.addBot(client)
	for _, call := range c.uBContext {

		//_, _ = call.App.UpdatesGetState()

		call.OnStreamEnd(func(chatID int64, streamType ntgcalls.StreamType, device ntgcalls.StreamDevice) {
			gologging.InfoF("[TelegramCalls] The stream has ended in chat %d (type=%v, device=%v)", chatID, streamType, device)
			if streamType == ntgcalls.VideoStream {
				gologging.DebugF("Ignoring video stream end for chat %d", chatID)
				return
			}

			if err := c.PlayNext(chatID); err != nil {
				gologging.WarnF("[OnStreamEnd] Failed to play the song: %v", err)
			}
		})

		call.OnIncomingCall(func(ub *ubot.Context, chatId int64) {
			_, _ = ub.App.SendMessage(chatId, "Are you calling me? Let me play a song for you...")
			msg, err := dl.GetMessage(c.bot, "https://t.me/FallenSongs/1295")
			if err != nil {
				gologging.InfoF("[OnIncomingCall] Failed to get the message: %v", err)
				return
			}

			filePath, err := msg.Download(&tg.DownloadOptions{FileName: filepath.Join(config.Conf.DownloadsDir, msg.File.Name)})
			if err != nil {
				gologging.InfoF("[OnIncomingCall] Failed to download the message: %v", err)
				return
			}

			err = c.PlayMedia(chatId, filePath, false, "")
			if err != nil {

				gologging.InfoF("[OnIncomingCall] Failed to play the media: %v", err)
				return
			}

			return
		})

		call.OnFrame(func(chatId int64, mode ntgcalls.StreamMode, device ntgcalls.StreamDevice, frames []ntgcalls.Frame) {
			gologging.DebugF("Received frames for chatId: %d, mode: %v, device: %v", chatId, mode, device)
		})

		_, _ = call.App.SendMessage(client.Me().Username, "/start")
		_, err := call.App.SendMessage(config.Conf.LoggerId, "UB has started.")
		if err != nil {
			gologging.InfoF("[TelegramCalls - SendMessage] Failed to send message: %v", err)
		}
	}
}
