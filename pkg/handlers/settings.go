package handlers

import (
	"fmt"
	"strings"
	"tgmusic/pkg/core"
	"tgmusic/pkg/core/cache"
	"tgmusic/pkg/core/db"

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

	// Get current settings
	playMode := db.Instance.GetPlayMode(ctx, chatID)
	adminMode := db.Instance.GetAdminMode(ctx, chatID)

	text := fmt.Sprintf("<b>⚙️ Settings for %s</b>\n\n<b>Play Mode:</b> %s\n<b>Admin Mode:</b> %s",
		m.Chat.Title, playMode, adminMode)

	_, err = m.Reply(text, telegram.SendOptions{
		ReplyMarkup: core.SettingsKeyboard(playMode, adminMode),
	})
	return err
}

func settingsCallbackHandler(c *telegram.CallbackQuery) error {
	chatID, err := getPeerId(c.Client, c.ChatID)
	if err != nil {
		gologging.WarnF("getPeerId error: %v", err)
		return nil
	}

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

	ctx, cancel := db.Ctx()
	defer cancel()

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
		gologging.WarnF("Failed to get chat: %v", err)
		return nil
	}

	text := fmt.Sprintf("<b>⚙️ Settings for %s</b>\n\n<b>Play Mode:</b> %s\n<b>Admin Mode:</b> %s",
		chat.Title, getPlayMode, getAdminMode)

	_, err = c.Edit(text, &telegram.SendOptions{
		ReplyMarkup: core.SettingsKeyboard(getPlayMode, getAdminMode),
	})
	if err != nil {
		gologging.WarnF("Failed to edit message: %v", err)
		return err
	}

	_, _ = c.Answer("✅ Settings updated", &telegram.CallbackOptions{Alert: false})
	return nil
}
