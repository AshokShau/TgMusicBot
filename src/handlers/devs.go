/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/config"
	"fmt"
	"strings"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

// activeVcHandler handles the /activevc command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func activeVcHandler(m *telegram.NewMessage) error {
	activeChats := cache.ChatCache.GetActiveChats()
	if len(activeChats) == 0 {
		_, err := m.Reply("No active chats found.")
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎵 <b>Active Voice Chats</b> (%d):\n\n", len(activeChats)))

	for _, chatID := range activeChats {
		queueLength := cache.ChatCache.GetQueueLength(chatID)
		currentSong := cache.ChatCache.GetPlayingTrack(chatID)

		var songInfo string
		if currentSong != nil {
			songInfo = fmt.Sprintf(
				"🎶 <b>Now Playing:</b> <a href='%s'>%s</a> (%ds)",
				currentSong.URL,
				currentSong.Name,
				currentSong.Duration,
			)
		} else {
			songInfo = "🔇 No song playing."
		}

		sb.WriteString(fmt.Sprintf(
			"➤ <b>Chat ID:</b> <code>%d</code>\n📌 <b>Queue Size:</b> %d\n%s\n\n",
			chatID,
			queueLength,
			songInfo,
		))
	}

	text := sb.String()
	if len(text) > 4096 {
		text = fmt.Sprintf("🎵 <b>Active Voice Chats</b> (%d)", len(activeChats))
	}

	_, err := m.Reply(text, &telegram.SendOptions{LinkPreview: false})
	if err != nil {
		return err
	}

	return nil
}

// Handles the /clearass command to remove all assistant assignments
func clearAssistantsHandler(m *telegram.NewMessage) error {
	ctx, cancel := db.Ctx()
	defer cancel()

	done, err := db.Instance.ClearAllAssistants(ctx)
	if err != nil {
		_, _ = m.Reply(fmt.Sprintf("failed to clear assistants: %s", err.Error()))
		return err
	}

	_, err = m.Reply(fmt.Sprintf("Removed assistant from %d chats", done))
	return err
}

// Handles the /leaveall command to leave all chats
func leaveAllHandler(m *telegram.NewMessage) error {
	reply, err := m.Reply("Assistant is leaving all chats...")
	if err != nil {
		return err
	}

	leftCount, err := vc.Calls.LeaveAll()
	if err != nil {
		_, _ = reply.Edit(fmt.Sprintf("Failed to leave all chats: %s", err.Error()))
		return err
	}

	_, err = reply.Edit(fmt.Sprintf("Assistant's Left %d chats", leftCount))
	return err
}

// Handles the /logger command to toggle logger status
func loggerHandler(m *telegram.NewMessage) error {
	ctx, cancel := db.Ctx()
	defer cancel()
	if config.Conf.LoggerId == 0 {
		_, _ = m.Reply("Please set LOGGER_ID in .env first.")
		return telegram.ErrEndGroup
	}

	loggerStatus := db.Instance.GetLoggerStatus(ctx, m.Client.Me().ID)
	args := strings.ToLower(m.Args())
	if len(args) == 0 {
		_, _ = m.Reply(fmt.Sprintf("Usage: /logger [enable|disable|on|off]\nCurrent status: %t", loggerStatus))
		return telegram.ErrEndGroup
	}

	switch args {
	case "enable", "on":
		_ = db.Instance.SetLoggerStatus(ctx, m.Client.Me().ID, true)
		_, _ = m.Reply("Logger Enabled")
	case "disable", "off":
		_ = db.Instance.SetLoggerStatus(ctx, m.Client.Me().ID, false)
		_, _ = m.Reply("Logger disabled")
	default:
		_, _ = m.Reply("Invalid argument. Use 'enable', 'disable', 'on', or 'off'.")
	}

	return telegram.ErrEndGroup
}
