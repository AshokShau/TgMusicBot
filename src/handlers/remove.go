/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"fmt"
	"strconv"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"

	"github.com/amarnathcjd/gogram/telegram"
)

// removeHandler handles the /remove command.
func removeHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.Reply(lang.GetString(langCode, "no_track_playing"))
		return nil
	}

	queue := cache.ChatCache.GetQueue(chatID)
	if len(queue) == 0 {
		_, _ = m.Reply(lang.GetString(langCode, "queue_empty"))
		return nil
	}

	args := m.Args()
	if args == "" {
		_, _ = m.Reply(lang.GetString(langCode, "remove_usage"))
		return nil
	}

	trackNum, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.Reply(lang.GetString(langCode, "remove_invalid_number"))
		return nil
	}

	if trackNum <= 0 || trackNum > len(queue) {
		_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "remove_out_of_range"), len(queue)))
		return nil
	}

	cache.ChatCache.RemoveTrack(chatID, trackNum)
	_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "remove_success"), trackNum, m.Sender.FirstName))
	return err
}
