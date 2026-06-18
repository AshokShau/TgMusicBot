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
func stopHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "The bot isn't streaming in the video chat.", nil)
		return nil
	}

	_ = vc.Calls.Stop(chatID, false)
	_, _ = m.ReplyText(c, fmt.Sprintf("<b>Stream ended by</b> %s", firstName(c, m)), replyOpts)
	return nil
}
