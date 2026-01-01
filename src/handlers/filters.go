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
	"ashokshau/tgmusic/src/utils"
	"strings"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"

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
	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()

	botStatus, err := cache.GetUserAdmin(m.Client, chatID, m.Client.Me().ID, false)
	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			_, _ = m.Reply("❌ bot is not admin in this chat.\nPlease promote me with Invite Users permission.")
			return false
		}

		logger.Warn("GetUserAdmin error: %v", err)
		_, _ = m.Reply("⚠️ Failed to get bot admin status (cache or fetch failed).")
		return false
	}

	if botStatus.Status != telegram.Admin && botStatus.Status != telegram.Creator {
		_, _ = m.Reply("❌ bot is not admin in this chat.\nUse /reload to refresh admin cache.")
		return false
	}

	if botStatus.Rights != nil && !botStatus.Rights.InviteUsers {
		_, _ = m.Reply("⚠️ bot doesn’t have permission to invite users.")
		return false
	}
	userID := m.SenderID()

	getAdminMode := db.Instance.GetAdminMode(ctx, chatID)
	if getAdminMode == utils.Everyone {
		return true
	}

	if getAdminMode == utils.Admins {
		if db.Instance.IsAdmin(ctx, chatID, userID) {
			return true
		}
		_, _ = m.Reply("❌ You are not an admin in this chat.")
		return false
	}

	if getAdminMode == utils.Auth {
		if db.Instance.IsAuthUser(ctx, chatID, userID) {
			return true
		}
		_, _ = m.Reply("❌ You are not an authorized user in this chat.")
		return false
	}

	_, _ = m.Reply("❌ You are not an authorized user in this chat.")
	return false
}

func adminModeCB(cb *telegram.CallbackQuery) bool {
	chatID := cb.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()

	botStatus, err := cache.GetUserAdmin(cb.Client, chatID, cb.Client.Me().ID, false)
	opts := &telegram.CallbackOptions{Alert: true}

	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			_, _ = cb.Answer("❌ bot is not admin in this chat.\nPlease promote me with Invite Users permission.", opts)
			return false
		}

		logger.Warn("GetUserAdmin error: %v", err)
		_, _ = cb.Answer("⚠️ Failed to get bot admin status (cache or fetch failed).", opts)
		return false
	}

	if botStatus.Status != telegram.Admin && botStatus.Status != telegram.Creator {
		_, _ = cb.Answer("❌ bot is not admin in this chat.\nUse /reload to refresh admin cache.", opts)
		return false
	}

	if botStatus.Rights != nil && !botStatus.Rights.InviteUsers {
		_, _ = cb.Answer("⚠️ bot doesn’t have permission to invite users.", opts)
		return false
	}
	userID := cb.SenderID

	getAdminMode := db.Instance.GetAdminMode(ctx, chatID)
	if getAdminMode == utils.Everyone {
		return true
	}

	if getAdminMode == utils.Admins {
		if db.Instance.IsAdmin(ctx, chatID, userID) {
			return true
		}
		_, _ = cb.Answer("❌ You are not an admin in this chat.", opts)
		return false
	}

	if getAdminMode == utils.Auth {
		if db.Instance.IsAuthUser(ctx, chatID, userID) {
			return true
		}
		_, _ = cb.Answer("❌ You are not an authorized user in this chat.", opts)
		return false
	}

	_, _ = cb.Answer("❌ You are not an authorized user in this chat.", opts)
	return false
}

func playMode(m *telegram.NewMessage) bool {
	if m.IsPrivate() {
		return false
	}

	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()

	botStatus, err := cache.GetUserAdmin(m.Client, chatID, m.Client.Me().ID, false)
	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			_, _ = m.Reply("❌ bot is not admin in this chat.\nPlease promote me with Invite Users permission.")
			return false
		}

		logger.Warn("GetUserAdmin error: %v", err)
		_, _ = m.Reply("⚠️ Failed to get bot admin status (cache or fetch failed).")
		return false
	}

	if botStatus.Status != telegram.Admin && botStatus.Status != telegram.Creator {
		_, _ = m.Reply("❌ bot is not admin in this chat.\nUse /reload to refresh admin cache.")
		return false
	}

	if botStatus.Rights != nil && !botStatus.Rights.InviteUsers {
		_, _ = m.Reply("⚠️ bot doesn’t have permission to invite users.")
		return false
	}

	getPlayMode := db.Instance.GetPlayMode(ctx, chatID)
	if getPlayMode != utils.Everyone {
		admins, err := cache.GetAdmins(m.Client, chatID, false)
		if err != nil {
			logger.Warn("getAdmins error: %v", err)
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
			if getPlayMode == utils.Auth {
				if !db.Instance.IsAuthUser(ctx, chatID, m.Sender.ID) {
					_, _ = m.Reply("You are not authorized to use this command.")
					return false
				}
			} else {
				_, _ = m.Reply("You are not authorized to use this command.")
				return false
			}
		}
	}

	return true
}
