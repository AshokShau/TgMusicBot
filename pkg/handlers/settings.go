package handlers

import (
	"fmt"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

func settingsHandler(m *telegram.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	ctx, cancel := db.Ctx()
	defer cancel()

	chatID, _ := getPeerId(m.Client, m.ChatID())
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
	langCode := db.Instance.GetLang(ctx, chatID)
	// Get current settings
	getPlayMode := db.Instance.GetPlayMode(ctx, chatID)
	getAdminMode := db.Instance.GetAdminMode(ctx, chatID)

	text := fmt.Sprintf(lang.GetString(langCode, "settings_header"),
		m.Chat.Title, getPlayMode, getAdminMode)

	_, err = m.Reply(text, telegram.SendOptions{
		ReplyMarkup: core.SettingsKeyboard(getPlayMode, getAdminMode),
	})
	return err
}

func settingsCallbackHandler(c *telegram.CallbackQuery) error {
	chatID, err := getPeerId(c.Client, c.ChatID)
	if err != nil {
		gologging.WarnF("getPeerId error: %v", err)
		return nil
	}
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

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
		_, err := c.Answer(lang.GetString(langCode, "settings_no_permission"), &telegram.CallbackOptions{Alert: true})
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
		cache.Admins:   true,
		cache.Auth:     true,
		cache.Everyone: true,
	}

	if !validValues[settingValue] {
		_, _ = c.Answer(lang.GetString(langCode, "settings_update_invalid"), &telegram.CallbackOptions{Alert: true})
		return nil
	}

	switch settingType {
	case "play":
		_ = db.Instance.SetPlayMode(ctx, chatID, settingValue)
	case "admin":
		_ = db.Instance.SetAdminMode(ctx, chatID, settingValue)
	default:
		_, _ = c.Answer(lang.GetString(langCode, "settings_update_prompt"), &telegram.CallbackOptions{Alert: true})
		return nil
	}

	// Get updated settings
	getPlayMode := db.Instance.GetPlayMode(ctx, chatID)
	getAdminMode := db.Instance.GetAdminMode(ctx, chatID)
	chat, err := c.GetChannel()
	if err != nil {
		gologging.WarnF("Failed to get chat: %v", err)
		return nil
	}

	text := fmt.Sprintf(lang.GetString(langCode, "settings_header"),
		chat.Title, getPlayMode, getAdminMode)

	_, err = c.Edit(text, &telegram.SendOptions{
		ReplyMarkup: core.SettingsKeyboard(getPlayMode, getAdminMode),
	})
	if err != nil {
		gologging.WarnF("Failed to edit message: %v", err)
		return err
	}

	_, _ = c.Answer(lang.GetString(langCode, "settings_updated"), &telegram.CallbackOptions{Alert: false})
	_, _ = c.Edit(lang.GetString(langCode, "settings_updated"))
	return nil
}
