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

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

// pauseHandler handles the /pause command.
func pauseHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.Reply("⏸ No track currently playing.")
		return nil
	}

	if _, err := vc.Calls.Pause(chatID); err != nil {
		_, _ = m.Reply(fmt.Sprintf("❌ An error occurred while pausing the playback: %s", err.Error()))
		return nil
	}

	_, err := m.Reply(fmt.Sprintf("⏸️ Playback has been paused by %s.", m.Sender.FirstName), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("pause")})
	return err
}

// resumeHandler handles the /resume command.
func resumeHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()

	if chatID > 0 {
		_, _ = m.Reply("This command can only be used in a supergroup.")
		return nil
	}

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.Reply("⏸ No track currently playing.")
		return nil
	}

	if _, err := vc.Calls.Resume(chatID); err != nil {
		_, _ = m.Reply(fmt.Sprintf("❌ An error occurred while resuming the playback: %s", err.Error()))
		return nil
	}

	_, err := m.Reply(fmt.Sprintf("▶️ Playback has been resumed by %s.", m.Sender.FirstName), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("resume")})
	return err
}
