/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"fmt"
	"strconv"

	"ashokshau/tgmusic/src/core/cache"

	td "github.com/AshokShau/gotdbot"
)

// removeHandler handles the /remove command.
func removeHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}
	
	chatID := ctx.EffectiveChatId
	m := ctx.EffectiveMessage

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "⏸ No track currently playing.", nil)
		return nil
	}

	queue := cache.ChatCache.GetQueue(chatID)
	if len(queue) == 0 {
		_, _ = m.ReplyText(c, "📭 The queue is currently empty.", nil)
		return nil
	}

	args := Args(m)
	if args == "" {
		_, _ = m.ReplyText(c, "<b>❌ Remove Track</b>\n\n<b>Usage:</b> <code>/remove [track number]</code>\n\n- Use <code>1</code> to remove the first track, <code>2</code> for the second, and so on.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return nil
	}

	trackNum, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.ReplyText(c, "⚠️ Please enter a valid track number.", nil)
		return nil
	}

	if trackNum <= 0 || trackNum > len(queue) {
		_, _ = m.ReplyText(c, fmt.Sprintf("⚠️ The track number is not valid. Please choose a number between 1 and %d.", len(queue)), nil)
		return nil
	}

	cache.ChatCache.RemoveTrack(chatID, trackNum)
	user, err := c.GetUser(m.SenderID())
	if err != nil {
		user = &td.User{FirstName: "Unknown"}
	}

	_, err = m.ReplyText(c, fmt.Sprintf("✅ Track #%d has been removed by %s.", trackNum, user.FirstName), nil)
	return err
}
