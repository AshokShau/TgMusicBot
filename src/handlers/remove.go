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

	"github.com/amarnathcjd/gogram/telegram"
)

// removeHandler handles the /remove command.
func removeHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.Reply("⏸ No track currently playing.")
		return nil
	}

	queue := cache.ChatCache.GetQueue(chatID)
	if len(queue) == 0 {
		_, _ = m.Reply("📭 The queue is currently empty.")
		return nil
	}

	args := m.Args()
	if args == "" {
		_, _ = m.Reply("<b>❌ Remove Track</b>\n\n<b>Usage:</b> <code>/remove [track number]</code>\n\n- Use <code>1</code> to remove the first track, <code>2</code> for the second, and so on.")
		return nil
	}

	trackNum, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.Reply("⚠️ Please enter a valid track number.")
		return nil
	}

	if trackNum <= 0 || trackNum > len(queue) {
		_, _ = m.Reply(fmt.Sprintf("⚠️ The track number is not valid. Please choose a number between 1 and %d.", len(queue)))
		return nil
	}

	cache.ChatCache.RemoveTrack(chatID, trackNum)
	_, err = m.Reply(fmt.Sprintf("✅ Track #%d has been removed by %s.", trackNum, m.Sender.FirstName))
	return err
}
