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

	td "github.com/AshokShau/gotdbot"
)

func settingsHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}

	m := ctx.EffectiveMessage
	ctx2, cancel := db.Ctx()
	defer cancel()

	chatID := ctx.EffectiveChatId
	admins, err := cache.GetAdmins(c, chatID, false)
	if err != nil {
		return err
	}

	// Check if user is admin
	var isAdmin bool
	for _, admin := range admins {
		if SenderID(admin.MemberId) == m.SenderID() {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		return nil
	}

	// Get current settings
	getPlayMode := db.Instance.GetPlayMode(ctx2, chatID)
	playModeStr := utils.Everyone
	if getPlayMode {
		playModeStr = utils.Admins
	}
	getAdminMode := db.Instance.GetAdminMode(ctx2, chatID)

	chat, err := m.GetChat(c)
	if err != nil {
		c.Logger.Warn("Failed to get chat", "error", err)
		return nil
	}

	text := fmt.Sprintf("<b>Settings for %s</b>\n\n<b>Play Mode:</b> %s\n<b>Admin Mode:</b> %s",
		chat.Title, playModeStr, getAdminMode)

	_, err = m.ReplyText(c, text, &td.SendTextMessageOpts{ReplyMarkup: core.SettingsKeyboard(playModeStr, getAdminMode), ParseMode: td.ParseModeHTML})
	return err
}

func settingsCallbackHandler(c *td.Client, ctx *td.Context) error {
	chatID := ctx.EffectiveChatId
	cb := ctx.Update.UpdateNewCallbackQuery

	ctx2, cancel := db.Ctx()
	defer cancel()

	// Check admin permissions
	admins, err := cache.GetAdmins(c, chatID, false)
	if err != nil {
		return err
	}

	var hasPerms bool
	for _, admin := range admins {
		if SenderID(admin.MemberId) == cb.SenderUserId {
			rights, _ := cache.GetRights(c, chatID, cb.SenderUserId, false)
			hasPerms = (rights != nil && rights.CanManageVideoChats) || admin.Status == td.ChatMemberStatusCreator{}
			break
		}
	}

	if !hasPerms {
		err = cb.Answer(c, 300, true, "You don't have permission to change settings.", "")
		return err
	}

	// Process the callback data
	parts := strings.Split(cb.DataString(), "_")
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
		_ = cb.Answer(c, 300, true, "Update your chat settings", "")
		return nil
	}

	switch settingType {
	case "play":
		adminPlay := settingValue == utils.Admins || settingValue == utils.Auth
		_ = db.Instance.SetPlayMode(ctx2, chatID, adminPlay)
	case "admin":
		_ = db.Instance.SetAdminMode(ctx2, chatID, settingValue)
	default:
		_ = cb.Answer(c, 300, true, "Update your chat settings", "")
		return nil
	}

	// Get updated settings
	getPlayMode := db.Instance.GetPlayMode(ctx2, chatID)
	playModeStr := utils.Everyone
	if getPlayMode {
		playModeStr = utils.Admins
	}
	getAdminMode := db.Instance.GetAdminMode(ctx2, chatID)
	chat, err := c.GetChat(chatID)
	if err != nil {
		c.Logger.Warn("Failed to get chat", "error", err)
		return nil
	}

	text := fmt.Sprintf("<b>Settings for %s</b>\n\n<b>Play Mode:</b> %s\n<b>Admin Mode:</b> %s",
		chat.Title, playModeStr, getAdminMode)

	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.SettingsKeyboard(playModeStr, getAdminMode), ParseMode: td.ParseModeHTML})
	if err != nil {
		return err
	}

	_ = cb.Answer(c, 200, false, "Settings updated", "")
	_, _ = cb.EditMessageText(c, text, nil)
	return nil
}
