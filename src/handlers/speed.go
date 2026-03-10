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
	"ashokshau/tgmusic/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// speedHandler handles the /speed command.
func speedHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}
	chatID := ctx.EffectiveChatId
	m := ctx.EffectiveMessage

	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "⏸ No track currently playing.", nil)
		return err
	}

	if playingSong := cache.ChatCache.GetPlayingTrack(chatID); playingSong == nil {
		_, err := m.ReplyText(c, "⏸ No track currently playing.", nil)
		return err
	}

	args := Args(m)
	if args == "" {
		_, _ = m.ReplyText(c, "<b>❌ Change Speed</b>\n\n<b>Usage:</b> <code>/speed [value]</code>\n\n- The speed can be set from <code>0.5</code> to <code>4.0</code>.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return nil
	}

	speed, err := strconv.ParseFloat(args, 64)
	if err != nil {
		_, _ = m.ReplyText(c, "❌ Invalid speed value provided. Please use a number between 0.5 and 4.0.", nil)
		return nil
	}

	if speed < 0.5 || speed > 4.0 {
		_, _ = m.ReplyText(c, "⚠️ The speed must be between 0.5 and 4.0.", nil)
		return nil
	}

	if err = vc.Calls.ChangeSpeed(chatID, speed); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ An error occurred while changing the speed: %s", err.Error()), nil)
		return nil
	}
	_, _ = m.ReplyText(c, fmt.Sprintf("✅ The playback speed has been changed to %.2fx.", speed), nil)
	return nil
}
