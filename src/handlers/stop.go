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

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// stopHandler handles the /stop command.
func stopHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}
	m := ctx.EffectiveMessage
	chatID := ctx.EffectiveChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "⏸ Nothing is playing.", nil)
		return nil
	}

	if err := vc.Calls.Stop(chatID); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ Error stopping playback: %s", err.Error()), nil)
		return err
	}

	user, err := c.GetUser(m.SenderID())
	if err != nil {
		user = &td.User{FirstName: "Unknown"}
	}

	_, _ = m.ReplyText(c, fmt.Sprintf("⏹️ Stopped by %s. Queue cleared.", user.FirstName), nil)
	return nil
}
