/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"
	"fmt"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func handleVoiceChatMessage(m *telegram.NewMessage) error {
	if m.Action == nil {
		return nil
	}

	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	client := m.Client
	defer cancel()

	langCode := "en"
	if db.Instance != nil {
		langCode = db.Instance.GetLang(ctx, chatID)
	}

	// Chat is not a Supergroup
	if m.Channel == nil {
		text := fmt.Sprintf(
			lang.GetString(langCode, "watcher_not_supergroup"),
			chatID,
		)

		_, _ = client.SendMessage(chatID, text, &telegram.SendOptions{
			ReplyMarkup: core.AddMeMarkup(client.Me().Username),
			LinkPreview: false,
		})

		time.Sleep(1 * time.Second)
		_ = client.LeaveChannel(chatID)
		return nil
	}

	action, ok := m.Action.(*telegram.MessageActionGroupCall)
	if !ok {
		return telegram.ErrEndGroup
	}

	var message string

	if action.Duration == 0 {
		cache.ChatCache.ClearChat(chatID)
		message = lang.GetString(langCode, "watcher_vc_started")
	} else {
		cache.ChatCache.ClearChat(chatID)
		logger.Info("Voice chat ended. Duration: %d seconds", action.Duration)
		message = lang.GetString(langCode, "watcher_vc_ended")
	}

	_, _ = m.Client.SendMessage(chatID, message)
	return telegram.ErrEndGroup
}
