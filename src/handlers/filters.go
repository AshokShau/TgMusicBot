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
	"strings"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"

	td "github.com/AshokShau/gotdbot"
)

// adminMode checks if the bot is an admin in the chat.
func adminMode(c *td.Client, ctx *td.Context) bool {
	m := ctx.EffectiveMessage
	if m.IsPrivate() {
		return false
	}

	chatID := m.ChatId
	ctx2, cancel := db.Ctx()
	defer cancel()

	botStatus, err := cache.GetUserAdmin(c, chatID, c.Me().Id, false)
	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			_, _ = m.ReplyText(c, "❌ bot is not admin in this chat.\nPlease promote me with Invite Users permission.", nil)
			return false
		}

		c.Logger.Warn("GetUserAdmin error", "error", err)
		_, _ = m.ReplyText(c, "⚠️ Failed to get bot admin status (cache or fetch failed).", nil)
		return false
	}

	switch s := botStatus.Status.(type) {
	case *td.ChatMemberStatusCreator:
		return true
	case *td.ChatMemberStatusAdministrator:
		if s.Rights == nil || !s.Rights.CanInviteUsers {
			_, _ = m.ReplyText(c, "⚠️ bot doesn’t have permission to invite users.", nil)
			return false
		}

	default:
		_, _ = m.ReplyText(c, "❌ bot is not admin in this chat.\nUse /reload to refresh admin cache.", nil)
		return false
	}

	userID := m.SenderID()
	getAdminMode := db.Instance.GetAdminMode(ctx2, chatID)
	if getAdminMode == utils.Everyone {
		return true
	}

	if getAdminMode == utils.Admins {
		if db.Instance.IsAdmin(ctx2, chatID, userID) {
			return true
		}
		_, _ = m.ReplyText(c, "❌ You are not an admin in this chat.", nil)
		return false
	}

	_, _ = m.ReplyText(c, "❌ You are not an authorized user in this chat.", nil)
	return false
}

func adminModeCB(c *td.Client, cb *td.UpdateNewCallbackQuery) bool {
	chatID := cb.ChatId
	ctx, cancel := db.Ctx()
	defer cancel()

	botStatus, err := cache.GetUserAdmin(c, chatID, c.Me().Id, false)
	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			_ = cb.Answer(c, 300, true, "❌ bot is not admin in this chat.\nPlease promote me with Invite Users permission.", "")
			return false
		}

		c.Logger.Warn("GetUserAdmin error", "error", err)
		_ = cb.Answer(c, 300, true, "⚠️ Failed to get bot admin status (cache or fetch failed).", "")
		return false
	}

	switch s := botStatus.Status.(type) {

	case *td.ChatMemberStatusCreator:
		// creator always has full permissions
		return true

	case *td.ChatMemberStatusAdministrator:
		if s.Rights == nil || !s.Rights.CanInviteUsers {
			_ = cb.Answer(c, 300, true, "⚠️ bot doesn’t have permission to invite users.", "")
			return false
		}

	default:
		_ = cb.Answer(c, 300, true, "❌ bot is not admin in this chat.\nUse /reload to refresh admin cache.", "")
		return false
	}
	userID := cb.SenderUserId

	getAdminMode := db.Instance.GetAdminMode(ctx, chatID)
	if getAdminMode == utils.Everyone {
		return true
	}

	if getAdminMode == utils.Admins {
		if db.Instance.IsAdmin(ctx, chatID, userID) {
			return true
		}
		_ = cb.Answer(c, 300, true, "❌ You are not an admin in this chat.", "")
		return false
	}

	_ = cb.Answer(c, 300, true, "❌ You are not an authorized user in this chat.", "")
	return false
}

func playMode(c *td.Client, ctx *td.Context) bool {
	m := ctx.EffectiveMessage
	if m.IsPrivate() {
		return false
	}

	chatID := m.ChatID()
	ctx2, cancel := db.Ctx()
	defer cancel()

	botStatus, err := cache.GetUserAdmin(c, chatID, c.Me().Id, false)
	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			_, _ = m.ReplyText(c, "❌ bot is not admin in this chat.\nPlease promote me with Invite Users permission.", nil)
			return false
		}

		c.Logger.Warn("GetUserAdmin error", "error", err)
		_, _ = m.ReplyText(c, "⚠️ Failed to get bot admin status (cache or fetch failed).", nil)
		return false
	}

	switch s := botStatus.Status.(type) {
	case *td.ChatMemberStatusCreator:
		return true
	case *td.ChatMemberStatusAdministrator:
		if s.Rights == nil || !s.Rights.CanInviteUsers {
			_, _ = m.ReplyText(c, "⚠️ bot doesn’t have permission to invite users.", nil)
			return false
		}
	default:
		_, _ = m.ReplyText(c, "❌ bot is not admin in this chat.\nUse /reload to refresh admin cache.", nil)
		return false
	}

	getPlayMode := db.Instance.GetPlayMode(ctx2, chatID)
	if getPlayMode {
		admins, err := cache.GetAdmins(c, chatID, false)
		if err != nil {
			c.Logger.Warn("getAdmins error", "error", err)
			return false
		}

		var isAdmin bool
		for _, admin := range admins {
			// check if sender is an admin in the chat
			if admin.MemberId == m.SenderId {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			if !db.Instance.IsAuthUser(ctx2, chatID, m.SenderID()) {
				_, _ = m.ReplyText(c, "You are not authorized to use this command.", nil)
				return false
			}
		}
	}

	return true
}
