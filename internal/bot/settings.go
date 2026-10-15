/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tgmusic/internal/cache"
	"ashokshau/tgmusic/internal/db"
	"ashokshau/tgmusic/internal/utils"
	"fmt"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func settingsHandler(c *td.Client, m *td.Message) error {
	if m.IsPrivate() {
		return nil
	}

	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId
	admins, err := cache.GetAdmins(c, chatID, false)
	if err != nil {
		return err
	}

	var admin bool
	for _, a := range admins {
		if SenderID(a.MemberId) == m.SenderID() {
			admin = true
			break
		}
	}

	if !admin {
		return nil
	}

	getPlayMode := db.Instance.GetPlayMode(chatID)
	playModeStr := utils.Everyone
	if getPlayMode {
		playModeStr = utils.Admins
	}
	getAdminMode := db.Instance.GetAdminMode(chatID)
	cmdDelete := db.Instance.GetCmdDelete(chatID)
	language, _ := db.Instance.GetLanguage(chatID)
	autoplay := cache.ChatCache.GetAutoplay(chatID)

	chat, err := m.GetChat(c)
	if err != nil {
		c.Logger.Warn("Failed to get chat", "error", err)
		return nil
	}

	text := fmt.Sprintf("<u><b>%s settings</b></u>\n\nClick the buttons below to change this chat's current settings.",
		chat.Title)

	_, err = m.ReplyText(c, text, &td.SendMessageOpts{ReplyMarkup: utils.SettingsKeyboard(playModeStr, getAdminMode, cmdDelete, language, autoplay), ParseMode: td.ParseModeHTML})
	return err
}

func settingsCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if cb.IsPrivate() {
		return nil
	}

	chatID := cb.ChatId

	admins, err := cache.GetAdmins(c, chatID, false)
	if err != nil {
		return err
	}

	var hasPerms bool
	for _, a := range admins {
		if SenderID(a.MemberId) == cb.SenderUserId {
			rights, _ := cache.GetRights(c, chatID, cb.SenderUserId, false)
			hasPerms = (rights != nil && rights.CanManageVideoChats) || a.Status == td.ChatMemberStatusCreator{}
			break
		}
	}

	if !hasPerms {
		err = cb.Answer(c, "You don't have permission to change settings.", &td.CallbackAnswerOpts{ShowAlert: true})
		return err
	}

	data := cb.DataString()
	if data == "settings_main" {
		return cb.Answer(c, "Update your chat settings", nil)
	}

	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		return nil
	}

	settingType := parts[1]

	switch settingType {
	case "delete":
		cmdDelete := db.Instance.GetCmdDelete(chatID)
		if !cmdDelete {
			rights, err := cache.GetRights(c, chatID, c.Me.Id, false)
			if err != nil || rights == nil || !rights.CanDeleteMessages {
				_ = cb.Answer(c, "I don't have permission to delete messages in this chat.", &td.CallbackAnswerOpts{ShowAlert: true})
				return nil
			}
			_ = db.Instance.SetCmdDelete(chatID, true)
		} else {
			_ = db.Instance.SetCmdDelete(chatID, false)
		}
	case "play":
		getPlayMode := db.Instance.GetPlayMode(chatID)
		_ = db.Instance.SetPlayMode(chatID, !getPlayMode)
	case "admin":
		getAdminMode := db.Instance.GetAdminMode(chatID)
		newMode := utils.Everyone
		if getAdminMode == utils.Everyone {
			newMode = utils.Admins
		}
		_ = db.Instance.SetAdminMode(chatID, newMode)
	case "autoplay":
		if cache.ChatCache.GetPlayingTrack(chatID) == nil {
			_ = cb.Answer(c, "Bot is not streaming in the video chat.", &td.CallbackAnswerOpts{ShowAlert: true})
			return nil
		}
		autoplay := cache.ChatCache.GetAutoplay(chatID)
		cache.ChatCache.SetAutoplay(chatID, !autoplay)
	case "lang":
		return cb.Answer(c, "Language selection is not yet implemented via this menu.", &td.CallbackAnswerOpts{ShowAlert: true})
	default:
		return cb.Answer(c, "Unknown setting", &td.CallbackAnswerOpts{ShowAlert: true})
	}

	getPlayMode := db.Instance.GetPlayMode(chatID)
	playModeStr := utils.Everyone
	if getPlayMode {
		playModeStr = utils.Admins
	}
	getAdminMode := db.Instance.GetAdminMode(chatID)
	cmdDelete := db.Instance.GetCmdDelete(chatID)
	language, _ := db.Instance.GetLanguage(chatID)
	autoplay := cache.ChatCache.GetAutoplay(chatID)

	chat, err := c.GetChat(chatID)
	if err != nil {
		c.Logger.Warn("Failed to get chat", "error", err)
		return nil
	}

	text := fmt.Sprintf("<u><b>%s settings</b></u>\n\nClick the buttons below to change this chat's current settings.",
		chat.Title)

	_, err = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: utils.SettingsKeyboard(playModeStr, getAdminMode, cmdDelete, language, autoplay), ParseMode: td.ParseModeHTML})
	if err != nil {
		return err
	}

	_ = cb.Answer(c, "Settings updated", nil)
	return nil
}
