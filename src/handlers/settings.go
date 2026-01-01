/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/utils"
	"fmt"
	"strings"

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"

	"github.com/amarnathcjd/gogram/telegram"
)

func settingsHandler(m *telegram.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	ctx, cancel := db.Ctx()
	defer cancel()

	chatID := m.ChannelID()
	admins, err := cache.GetAdmins(m.Client, chatID, false)
	if err != nil {
		return err
	}

	// Check if user is admin
	var isAdmin bool
	for _, admin := range admins {
		if admin.User.ID == m.Sender.ID {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		return nil
	}
	// Get current settings
	getPlayMode := db.Instance.GetPlayMode(ctx, chatID)
	getAdminMode := db.Instance.GetAdminMode(ctx, chatID)

	text := fmt.Sprintf("<b>Settings for %s</b>\n\n<b>Play Mode:</b> %s\n<b>Admin Mode:</b> %s",
		m.Chat.Title, getPlayMode, getAdminMode)

	_, err = m.Reply(text, &telegram.SendOptions{
		ReplyMarkup: core.SettingsKeyboard(getPlayMode, getAdminMode),
	})
	return err
}

func settingsCallbackHandler(c *telegram.CallbackQuery) error {
	chatID := c.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()

	// Check admin permissions
	admins, err := cache.GetAdmins(c.Client, chatID, false)
	if err != nil {
		return err
	}

	var hasPerms bool
	for _, admin := range admins {
		if admin.User.ID == c.Sender.ID {
			hasPerms = (admin.Rights != nil && admin.Rights.ManageCall) || admin.Status == telegram.Creator
			break
		}
	}

	if !hasPerms {
		_, err := c.Answer("You don't have permission to change settings.", &telegram.CallbackOptions{Alert: true})
		return err
	}

	// Process the callback data
	parts := strings.Split(c.DataString(), "_")
	if len(parts) < 3 {
		return nil
	}

	// Update the appropriate setting
	settingType := parts[1]
	settingValue := parts[2]

	// Validate the setting value
	validValues := map[string]bool{
		utils.Admins:   true,
		utils.Auth:     true,
		utils.Everyone: true,
	}

	if !validValues[settingValue] {
		_, _ = c.Answer("Update your chat settings", &telegram.CallbackOptions{Alert: true})
		return nil
	}

	switch settingType {
	case "play":
		_ = db.Instance.SetPlayMode(ctx, chatID, settingValue)
	case "admin":
		_ = db.Instance.SetAdminMode(ctx, chatID, settingValue)
	default:
		_, _ = c.Answer("Update your chat settings", &telegram.CallbackOptions{Alert: true})
		return nil
	}

	// Get updated settings
	getPlayMode := db.Instance.GetPlayMode(ctx, chatID)
	getAdminMode := db.Instance.GetAdminMode(ctx, chatID)
	chat, err := c.GetChannel()
	if err != nil {
		logger.Warn("Failed to get chat: %v", err)
		return nil
	}

	text := fmt.Sprintf("<b>Settings for %s</b>\n\n<b>Play Mode:</b> %s\n<b>Admin Mode:</b> %s",
		chat.Title, getPlayMode, getAdminMode)

	_, err = c.Edit(text, &telegram.SendOptions{
		ReplyMarkup: core.SettingsKeyboard(getPlayMode, getAdminMode),
	})
	if err != nil {
		logger.Warn("Failed to edit message: %v", err)
		return err
	}

	_, _ = c.Answer("✅ Settings updated", &telegram.CallbackOptions{Alert: false})
	_, _ = c.Edit("✅ Settings updated")
	return nil
}
