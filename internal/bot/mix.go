/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tgmusic/internal/cache"
	"ashokshau/tgmusic/internal/config"
	"ashokshau/tgmusic/internal/downloader"
	"ashokshau/tgmusic/internal/utils"
	"context"
	"fmt"
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
)

func mixHandler(c *td.Client, m *td.Message) error {
	if !playMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if queueLen := cache.ChatCache.GetQueueLength(chatID); queueLen >= 10 {
		_, _ = m.ReplyText(c, "Queue is full (max 10 tracks). Use /end to clear.", nil)
		return td.EndGroups
	}

	isReply := m.ReplyToMessageID() != 0
	args := Args(m)
	source := m
	if isReply {
		if replied, err := getRepliedMessage(c, m); err == nil {
			source = replied
		}
	}
	url := getUrl(source)
	if isReply && source != m && args == "" && url == "" {
		args = source.Text()
	}

	input := coalesce(url, args)
	targetURL := ""

	if input == "" {
		playing := cache.ChatCache.GetPlayingTrack(chatID)
		if playing == nil {
			_, _ = m.ReplyText(c, "<b>Usage:</b>\n/mix [song or URL]\n\nOr run /mix while a song is currently playing to create a mix based on it.", &td.SendMessageOpts{ParseMode: "HTML"})
			return td.EndGroups
		}
		targetURL = playing.URL
		if playing.Platform == utils.Telegram || playing.Platform == utils.DirectLink {
			targetURL = strings.TrimSpace(playing.Name + " " + playing.Channel)
		}
	} else {
		targetURL = input
	}

	updater, err := m.ReplyText(c, "Fetching mix tracks...", nil)
	if err != nil {
		c.Logger.Warn("failed to send message", "error", err)
		return td.EndGroups
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	limit := int(config.AutoPlayLimit)
	tracks, err := downloader.GetRecommendations(ctx, targetURL, limit)
	if err != nil {
		c.Logger.Errorf("load mix recommendations in chat %d: %v", chatID, err)
		_, editErr := updater.EditText(c, "Unable to get a mix right now. Please try again later.", nil)
		if editErr != nil {
			return fmt.Errorf("edit mix error in chat %d: %w", chatID, editErr)
		}
		return nil
	}

	if len(tracks) == 0 {
		_, err = updater.EditText(c, "No tracks found for mix.", nil)
		if err != nil {
			return fmt.Errorf("edit empty mix result in chat %d: %w", chatID, err)
		}
		return nil
	}

	var filtered []utils.GetUrlTrack
	for _, track := range tracks {
		if _track := cache.ChatCache.GetTrackIfExists(chatID, track.Id); _track == nil {
			filtered = append(filtered, track)
		}
	}

	if len(filtered) == 0 {
		_, err = updater.EditText(c, "All mix tracks are already in the queue or playing.", nil)
		if err != nil {
			return fmt.Errorf("edit duplicate mix result in chat %d: %w", chatID, err)
		}
		return nil
	}

	return handleMultipleTracks(c, m, updater, filtered, chatID, false, false)
}
