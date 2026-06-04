/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package vc

import (
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/dl"
	"ashokshau/tgmusic/src/utils"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
	"github.com/amarnathcjd/gogram/telegram"
)

// downloadAndPrepareSong handles the download and preparation of a song for playback.
// It returns an error if the download or preparation fails.
func (c *TelegramCalls) downloadAndPrepareSong(song *utils.CachedTrack, reply *td.Message) error {
	if song.FilePath != "" {
		return nil
	}

	dlPath, err := dl.DownloadCachedTrack(song, c.bot)
	song.FilePath = dlPath
	if err != nil || song.FilePath == "" {
		_, _ = reply.EditText(c.bot, "⚠️ Download failed. Skipping track...", nil)
		return err
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

// handleFlood manages flood wait errors by pausing execution for short waits.
// It sleeps only if the wait is <= 5 seconds. Otherwise, it returns false.
func handleFlood(err error) bool {
	wait := telegram.GetFloodWait(err)
	if wait <= 0 {
		return false
	}

	if wait > 5 {
		logger.Warn("Flood wait too long, skipping sleep", "seconds", wait)
		return false
	}

	logger.Warn("Flood wait detected, sleeping", "seconds", wait)
	time.Sleep(time.Duration(wait+1) * time.Second)
	return true
}

func getVideoDimensions(filePath string) (int, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", filePath)
	out, err := cmd.Output()
	if err != nil {
		logger.Warn("[getVideoDimensions] Failed to get video dimensions (%s): %v", filePath, err)
		return 0, 0
	}
	dimensions := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(dimensions) != 2 {
		logger.Warn("[getVideoDimensions] Invalid video dimensions(%s): %s", filePath, string(out))
		return 0, 0
	}

	width, _ := strconv.Atoi(dimensions[0])
	height, _ := strconv.Atoi(dimensions[1])
	return width, height
}

// UpdateMembership updates the membership status of a user in a specific chat.
func (c *TelegramCalls) UpdateMembership(chatId, userId int64, status td.ChatMemberStatus) {
	cacheKey := fmt.Sprintf("%d:%d", chatId, userId)
	if c.statusCache != nil {
		c.statusCache.Set(cacheKey, status)
		logger.Info("[UpdateMembership] The cache has been updated: chat= user= status=", "chat_id", chatId, "user_id", userId, "arg3", status)
	}
}

// UpdateInviteLink updates the invite link for a specific chat.
func (c *TelegramCalls) UpdateInviteLink(chatId int64, link string) {
	cacheKey := strconv.FormatInt(chatId, 10)
	if link == "" {
		c.inviteCache.Delete(cacheKey)
		return
	}
	c.inviteCache.Set(cacheKey, link)
}
