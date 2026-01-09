/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/config"
	"fmt"
	"strings"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"
	"ashokshau/tgmusic/src/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

// activeVcHandler handles the /activevc command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func activeVcHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
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

	_, err := m.Reply(text, &telegram.SendOptions{LinkPreview: false})
	if err != nil {
		return err
	}

	return nil
}

// Handles the /clearass command to remove all assistant assignments
func clearAssistantsHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
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
	chatID := m.ChannelID()
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

// cleanupChatsHandler handles the /cleanupchats command.
// It removes chat IDs with invalid format (like -207... instead of -100...)
func cleanupChatsHandler(m *telegram.NewMessage) error {
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, m.ChannelID())

	args := strings.ToLower(m.Args())
	previewMode := strings.Contains(args, "-preview") || strings.Contains(args, "-dry")

	chats, err := db.Instance.GetAllChats(ctx)
	if err != nil {
		_, _ = m.Reply(fmt.Sprintf("❌ Failed to get chats: %v", err))
		return telegram.ErrEndGroup
	}

	var invalidChats []int64
	var validChats []int64

	for _, chatID := range chats {
		// Valid supergroup/channel IDs should be < -1000000000000 (e.g., -1001234567890)
		// Invalid IDs like -2072413383014 are > -3000000000000 but < -2000000000000
		// This catches IDs that don't have proper -100 prefix
		if chatID < -2000000000000 && chatID > -3000000000000 {
			invalidChats = append(invalidChats, chatID)
		} else if chatID < 0 {
			validChats = append(validChats, chatID)
		}
	}

	if len(invalidChats) == 0 {
		_, _ = m.Reply(fmt.Sprintf(
			"✅ <b>No Invalid Chats Found</b>\n\n"+
				"📊 Total chats: <code>%d</code>\n"+
				"✓ All chat IDs have valid format.",
			len(chats),
		))
		return telegram.ErrEndGroup
	}

	// Preview mode - just show what would be deleted
	if previewMode {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(
			"🔍 <b>Preview Mode - Invalid Chats Found</b>\n\n"+
				"📊 Total chats: <code>%d</code>\n"+
				"❌ Invalid chats: <code>%d</code>\n"+
				"✓ Valid chats: <code>%d</code>\n\n"+
				"<b>Invalid Chat IDs:</b>\n",
			len(chats), len(invalidChats), len(validChats),
		))

		// Show up to 20 invalid IDs
		showCount := len(invalidChats)
		if showCount > 20 {
			showCount = 20
		}
		for i := 0; i < showCount; i++ {
			sb.WriteString(fmt.Sprintf("• <code>%d</code>\n", invalidChats[i]))
		}
		if len(invalidChats) > 20 {
			sb.WriteString(fmt.Sprintf("... and %d more\n", len(invalidChats)-20))
		}

		sb.WriteString("\n💡 Run <code>/cleanupchats</code> without -preview to delete these.")

		text := sb.String()
		if len(text) > 4096 {
			text = fmt.Sprintf(
				"🔍 <b>Preview Mode</b>\n\n"+
					"📊 Total chats: <code>%d</code>\n"+
					"❌ Invalid chats: <code>%d</code>\n"+
					"✓ Valid chats: <code>%d</code>\n\n"+
					"💡 Run <code>/cleanupchats</code> to delete invalid chats.",
				len(chats), len(invalidChats), len(validChats),
			)
		}

		_, _ = m.Reply(text)
		return telegram.ErrEndGroup
	}

	// Delete invalid chats
	reply, _ := m.Reply(fmt.Sprintf("🧹 Cleaning up %d invalid chats...", len(invalidChats)))

	var removed int
	var failed int
	for _, chatID := range invalidChats {
		dbCtx, dbCancel := db.Ctx()
		if err := db.Instance.RemoveChat(dbCtx, chatID); err != nil {
			failed++
			logger.Warn("[Cleanup] Failed to remove chat %d: %v", chatID, err)
		} else {
			removed++
		}
		dbCancel()
	}

	result := fmt.Sprintf(
		"🧹 <b>Cleanup Complete</b>\n\n"+
			"📊 Total processed: <code>%d</code>\n"+
			"✅ Removed: <code>%d</code>\n"+
			"❌ Failed: <code>%d</code>\n"+
			"📁 Remaining valid chats: <code>%d</code>",
		len(invalidChats), removed, failed, len(validChats),
	)

	if reply != nil {
		_, _ = reply.Edit(result)
	} else {
		_, _ = m.Reply(result)
	}

	_ = langCode // Suppress unused variable warning
	return telegram.ErrEndGroup
}
