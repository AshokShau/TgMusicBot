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

	"github.com/amarnathcjd/gogram/telegram"
)

// stopHandler handles the /stop command.
func stopHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.Reply("⏸ Nothing is playing.")
		return nil
	}

	if err := vc.Calls.Stop(chatID); err != nil {
		_, _ = m.Reply(fmt.Sprintf("❌ Error stopping playback: %s", err.Error()))
		return err
	}

	_, _ = m.Reply(fmt.Sprintf("⏹️ Stopped by %s. Queue cleared.", m.Sender.FirstName))
	return nil
}
