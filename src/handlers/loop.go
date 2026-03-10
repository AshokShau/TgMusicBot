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

// loopHandler handles the /loop command.
func loopHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}

	m := ctx.EffectiveMessage
	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "⏸ No track currently playing.", nil)
		return err
	}

	args := Args(m)
	if args == "" {
		_, err := m.ReplyText(c, "<b>🔁 Loop Control</b>\n\n<b>Usage:</b> <code>/loop [count]</code>\n• <code>0</code> to disable loop\n• <code>1-10</code> to set the loop count", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	argsInt, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.ReplyText(c, "❌ Invalid loop count provided. Please use a number between 0 and 10.", nil)
		return nil
	}

	if argsInt < 0 || argsInt > 10 {
		_, err = m.ReplyText(c, "⚠️ The loop count must be between 0 and 10.", nil)
		return err
	}

	cache.ChatCache.SetLoopCount(chatID, argsInt)
	var action string
	if argsInt == 0 {
		action = "Looping has been disabled"
	} else {
		action = fmt.Sprintf("The loop has been set to %d time(s)", argsInt)
	}

	user, err := c.GetUser(m.SenderID())
	if err != nil {
		user = &td.User{FirstName: "Unknown"}
	}

	_, err = m.ReplyText(c, fmt.Sprintf("🔁 %s.\n\n└ Changed by: %s", action, user.FirstName), nil)
	return err
}
