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

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

// muteHandler handles the /mute command.
func muteHandler(m *telegram.NewMessage) error {
	if args := m.Args(); args != "" {
		return telegram.ErrEndGroup
	}

	chatID := m.ChannelID()
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.Reply("⏸ No track currently playing.")
		return err
	}

	if _, err := vc.Calls.Mute(chatID); err != nil {
		_, err = m.Reply(fmt.Sprintf("❌ An error occurred while muting the playback: %s", err.Error()))
		return err
	}

	_, err := m.Reply(fmt.Sprintf("🔇 Playback has been muted by %s.", m.Sender.FirstName), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("mute")})
	return err
}

// unmuteHandler handles the /unmute command.
func unmuteHandler(m *telegram.NewMessage) error {
	if args := m.Args(); args != "" {
		return telegram.ErrEndGroup
	}

	chatID := m.ChannelID()
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.Reply("⏸ No track currently playing.")
		return err
	}

	if _, err := vc.Calls.Unmute(chatID); err != nil {
		_, _ = m.Reply(fmt.Sprintf("❌ An error occurred while unmuting the playback: %s", err.Error()))
		return err
	}

	_, err := m.Reply(fmt.Sprintf("🔊 Playback has been unmuted by %s.", m.Sender.FirstName), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("unmute")})
	return err
}
