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
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"
	"ashokshau/tgmusic/src/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

// muteHandler handles the /mute command.
func muteHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.Reply(lang.GetString(langCode, "no_track_playing"))
		return err
	}

	if _, err := vc.Calls.Mute(chatID); err != nil {
		_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "mute_error"), err.Error()))
		return err
	}

	_, err := m.Reply(fmt.Sprintf(lang.GetString(langCode, "mute_success"), m.Sender.FirstName), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("mute")})
	return err
}

// unmuteHandler handles the /unmute command.
func unmuteHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.Reply(lang.GetString(langCode, "no_track_playing"))
		return err
	}

	if _, err := vc.Calls.Unmute(chatID); err != nil {
		_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "unmute_error"), err.Error()))
		return err
	}

	_, err := m.Reply(fmt.Sprintf(lang.GetString(langCode, "unmute_success"), m.Sender.FirstName), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("unmute")})
	return err
}
