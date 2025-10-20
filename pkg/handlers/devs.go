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
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"
	"github.com/AshokShau/TgMusicBot/pkg/vc"
	"github.com/amarnathcjd/gogram/telegram"
)

// activeVcHandler handles the /activevc command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func activeVcHandler(m *telegram.NewMessage) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	activeChats := cache.ChatCache.GetActiveChats()
	if len(activeChats) == 0 {
		_, err := m.Reply(lang.GetString(langCode, "no_active_chats"))
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "active_chats_header"), len(activeChats)))

	for _, chatID := range activeChats {
		queueLength := cache.ChatCache.GetQueueLength(chatID)
		currentSong := cache.ChatCache.GetPlayingTrack(chatID)

		var songInfo string
		if currentSong != nil {
			songInfo = fmt.Sprintf(
				lang.GetString(langCode, "now_playing_devs"),
				currentSong.URL,
				currentSong.Name,
				currentSong.Duration,
			)
		} else {
			songInfo = lang.GetString(langCode, "no_song_playing")
		}

		sb.WriteString(fmt.Sprintf(
			lang.GetString(langCode, "chat_info"),
			chatID,
			queueLength,
			songInfo,
		))
	}

	text := sb.String()
	if len(text) > 4096 {
		text = fmt.Sprintf(lang.GetString(langCode, "active_chats_header_short"), len(activeChats))
	}

	_, err := m.Reply(text, telegram.SendOptions{LinkPreview: false})
	if err != nil {
		return err
	}

	return nil
}

// Handles the /clearass command to remove all assistant assignments
func clearAssistantsHandler(m *telegram.NewMessage) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	done, err := db.Instance.ClearAllAssistants(ctx)
	if err != nil {
		_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "clear_assistants_error"), err.Error()))
		return err
	}

	_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "clear_assistants_success"), done))
	return err
}

// Handles the /leaveall command to leave all chats
func leaveAllHandler(m *telegram.NewMessage) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	reply, err := m.Reply(lang.GetString(langCode, "leave_all_start"))
	if err != nil {
		return err
	}

	leftCount, err := vc.Calls.LeaveAll()
	if err != nil {
		_, _ = reply.Edit(fmt.Sprintf(lang.GetString(langCode, "leave_all_error"), err.Error()))
		return err
	}

	_, err = reply.Edit(fmt.Sprintf(lang.GetString(langCode, "leave_all_success"), leftCount))
	return err
}
