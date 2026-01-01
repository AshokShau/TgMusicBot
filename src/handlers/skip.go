/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

// skipHandler handles the /skip command.
func skipHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.Reply("⏸ Nothing is playing.")
		return nil
	}

	_ = vc.Calls.PlayNext(chatID)
	return nil
}
