package handlers

import (
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/config"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

// isDev checks if the user is a developer.
// It takes a telegram.NewMessage object as input.
// It returns true if the user is a developer, otherwise false.
func isDev(m *telegram.NewMessage) bool {
	for _, dev := range config.Conf.DEVS {
		if dev == m.SenderID() {
			return true
		}
	}
	return false
}

// adminMode checks if the bot is an admin in the chat.
// It takes a telegram.NewMessage object as input.
// It checks if the bot is an admin in the chat.
// Handle Admin Mode
// It returns true if the bot is an admin, otherwise false.
func adminMode(m *telegram.NewMessage) bool {
	if m.IsPrivate() {
		return false
	}
	chatID, err := getPeerId(m.Client, m.ChatID())
	if err != nil {
		gologging.WarnF("getPeerId error: %v", err)
		return false
	}
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	botStatus, err := cache.GetUserAdmin(m.Client, chatID, m.Client.Me().ID, false)
	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			_, _ = m.Reply(lang.GetString(langCode, "filter_bot_not_admin"))
			return false
		}

		gologging.WarnF("GetUserAdmin error: %v", err)
		_, _ = m.Reply(lang.GetString(langCode, "filter_bot_admin_status_failed"))
		return false
	}

	if botStatus.Status != telegram.Admin && botStatus.Status != telegram.Creator {
		_, _ = m.Reply(lang.GetString(langCode, "filter_bot_not_admin_reload"))
		return false
	}

	if botStatus.Rights != nil && !botStatus.Rights.InviteUsers {
		_, _ = m.Reply(lang.GetString(langCode, "filter_bot_no_invite_permission"))
		return false
	}
	userID := m.SenderID()

	getAdminMode := db.Instance.GetAdminMode(ctx, chatID)
	if getAdminMode == cache.Everyone {
		return true
	}

	if getAdminMode == cache.Admins {
		if db.Instance.IsAdmin(ctx, chatID, userID) {
			return true
		}
		_, _ = m.Reply(lang.GetString(langCode, "filter_not_admin"))
		return false
	}

	if getAdminMode == cache.Auth {
		if db.Instance.IsAuthUser(ctx, chatID, userID) {
			return true
		}
		_, _ = m.Reply(lang.GetString(langCode, "filter_not_authorized"))
		return false
	}

	_, _ = m.Reply(lang.GetString(langCode, "filter_not_authorized"))
	return false
}

func adminModeCB(cb *telegram.CallbackQuery) bool {
	chatID, err := getPeerId(cb.Client, cb.ChatID)
	if err != nil {
		gologging.WarnF("getPeerId error: %v", err)
		return false
	}
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	botStatus, err := cache.GetUserAdmin(cb.Client, chatID, cb.Client.Me().ID, false)
	opts := &telegram.CallbackOptions{Alert: true}

	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			_, _ = cb.Answer(lang.GetString(langCode, "filter_bot_not_admin"), opts)
			return false
		}

		gologging.WarnF("GetUserAdmin error: %v", err)
		_, _ = cb.Answer(lang.GetString(langCode, "filter_bot_admin_status_failed"), opts)
		return false
	}

	if botStatus.Status != telegram.Admin && botStatus.Status != telegram.Creator {
		_, _ = cb.Answer(lang.GetString(langCode, "filter_bot_not_admin_reload"), opts)
		return false
	}

	if botStatus.Rights != nil && !botStatus.Rights.InviteUsers {
		_, _ = cb.Answer(lang.GetString(langCode, "filter_bot_no_invite_permission"), opts)
		return false
	}
	userID := cb.SenderID

	getAdminMode := db.Instance.GetAdminMode(ctx, chatID)
	if getAdminMode == cache.Everyone {
		return true
	}

	if getAdminMode == cache.Admins {
		if db.Instance.IsAdmin(ctx, chatID, userID) {
			return true
		}
		_, _ = cb.Answer(lang.GetString(langCode, "filter_not_admin"), opts)
		return false
	}

	if getAdminMode == cache.Auth {
		if db.Instance.IsAuthUser(ctx, chatID, userID) {
			return true
		}
		_, _ = cb.Answer(lang.GetString(langCode, "filter_not_authorized"), opts)
		return false
	}

	_, _ = cb.Answer(lang.GetString(langCode, "filter_not_authorized"), opts)
	return false
}

func playMode(m *telegram.NewMessage) bool {
	if m.IsPrivate() {
		return false
	}
	chatID, err := getPeerId(m.Client, m.ChatID())
	if err != nil {
		gologging.WarnF("getPeerId error: %v", err)
		return false
	}
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	botStatus, err := cache.GetUserAdmin(m.Client, chatID, m.Client.Me().ID, false)
	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			_, _ = m.Reply(lang.GetString(langCode, "filter_bot_not_admin"))
			return false
		}

		gologging.WarnF("GetUserAdmin error: %v", err)
		_, _ = m.Reply(lang.GetString(langCode, "filter_bot_admin_status_failed"))
		return false
	}

	if botStatus.Status != telegram.Admin && botStatus.Status != telegram.Creator {
		_, _ = m.Reply(lang.GetString(langCode, "filter_bot_not_admin_reload"))
		return false
	}

	if botStatus.Rights != nil && !botStatus.Rights.InviteUsers {
		_, _ = m.Reply(lang.GetString(langCode, "filter_bot_no_invite_permission"))
		return false
	}
	getPlayMode := db.Instance.GetPlayMode(ctx, chatID)
	if getPlayMode != cache.Everyone {
		admins, err := cache.GetAdmins(m.Client, chatID, false)
		if err != nil {
			gologging.WarnF("getAdmins error: %v", err)
			return false
		}

		var isAdmin bool
		for _, admin := range admins {
			if admin.User.ID == m.Sender.ID {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			if getPlayMode == cache.Auth {
				if !db.Instance.IsAuthUser(ctx, chatID, m.Sender.ID) {
					_, _ = m.Reply(lang.GetString(langCode, "filter_not_authorized_command"))
					return false
				}
			} else {
				_, _ = m.Reply(lang.GetString(langCode, "filter_not_authorized_command"))
				return false
			}
		}
	}

	return true
}
