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

// pauseHandler handles the /pause command.
func pauseHandler(m *telegram.NewMessage) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.Reply(lang.GetString(langCode, "no_track_playing"))
		return nil
	}

	if _, err := vc.Calls.Pause(chatID); err != nil {
		_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "pause_error"), err.Error()))
		return nil
	}

	_, err := m.Reply(fmt.Sprintf(lang.GetString(langCode, "pause_success"), m.Sender.FirstName), telegram.SendOptions{ReplyMarkup: core.ControlButtons("pause")})
	return err
}

// resumeHandler handles the /resume command.
func resumeHandler(m *telegram.NewMessage) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if chatID > 0 {
		_, _ = m.Reply(lang.GetString(langCode, "supergroup_command_only"))
		return nil
	}

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.Reply(lang.GetString(langCode, "no_track_playing"))
		return nil
	}

	if _, err := vc.Calls.Resume(chatID); err != nil {
		_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "resume_error"), err.Error()))
		return nil
	}

	_, err := m.Reply(fmt.Sprintf(lang.GetString(langCode, "resume_success"), m.Sender.FirstName), telegram.SendOptions{ReplyMarkup: core.ControlButtons("resume")})
	return err
}
