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
	"slices"
	"strings"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"

	td "github.com/AshokShau/gotdbot"
)

// checkBotAdmin verifies the bot is an admin with CanInviteUsers permission.
// Returns true if the bot is owner/admin with required rights, false otherwise.
func checkBotAdmin(c *td.Client, chatID int64, replyErr func(msg string)) bool {
	botStatus, err := cache.GetUserAdmin(c, chatID, c.Me.Id, false)
	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			replyErr("❌ Bot is not an admin in this chat.\nPlease promote me with Invite Users permission.")
		} else {
			c.Logger.Warn("GetUserAdmin error", "error", err)
			replyErr("⚠️ Failed to get bot admin status.")
		}
		return false
	}

	switch s := botStatus.Status.(type) {
	case *td.ChatMemberStatusCreator:
		return true
	case *td.ChatMemberStatusAdministrator:
		if s.Rights == nil || !s.Rights.CanInviteUsers {
			replyErr("⚠️ Bot doesn't have permission to invite users.")
			return false
		}
		return true
	default:
		replyErr("❌ Bot is not an admin in this chat.\nUse /reload to refresh admin cache.")
		return false
	}
}

// adminMode checks if the bot is an admin in the chat and enforces admin mode restrictions.
func adminMode(c *td.Client, ctx *td.Context) bool {
	m := ctx.EffectiveMessage
	if m.IsPrivate() {
		return false
	}

	chatID := m.ChatId
	dbCtx, cancel := db.Ctx()
	defer cancel()

	if !checkBotAdmin(c, chatID, func(msg string) { _, _ = m.ReplyText(c, msg, nil) }) {
		return false
	}

	userID := m.SenderID()
	switch db.Instance.GetAdminMode(dbCtx, chatID) {
	case utils.Everyone:
		return true
	case utils.Admins:
		if db.Instance.IsAdmin(dbCtx, chatID, userID) || db.Instance.IsAuthUser(dbCtx, chatID, userID) {
			return true
		}
		_, _ = m.ReplyText(c, "❌ You are not an admin in this chat.", nil)
		return false
	default:
		_, _ = m.ReplyText(c, "❌ You are not an authorized user in this chat.", nil)
		return false
	}
}

func adminModeCB(c *td.Client, cb *td.UpdateNewCallbackQuery) bool {
	chatID := cb.ChatId
	dbCtx, cancel := db.Ctx()
	defer cancel()

	if !checkBotAdmin(c, chatID, func(msg string) { _ = cb.Answer(c, 300, true, msg, "") }) {
		return false
	}

	userID := cb.SenderUserId
	switch db.Instance.GetAdminMode(dbCtx, chatID) {
	case utils.Everyone:
		return true
	case utils.Admins:
		if db.Instance.IsAdmin(dbCtx, chatID, userID) || db.Instance.IsAuthUser(dbCtx, chatID, userID) {
			return true
		}
		_ = cb.Answer(c, 300, true, "❌ You are not an admin in this chat.", "")
		return false
	default:
		_ = cb.Answer(c, 300, true, "❌ You are not an authorized user in this chat.", "")
		return false
	}
}

// playMode checks if the bot is an admin and enforces play mode restrictions.
func playMode(c *td.Client, ctx *td.Context) bool {
	m := ctx.EffectiveMessage
	if m.IsPrivate() {
		return false
	}

	chatID := m.ChatID()
	dbCtx, cancel := db.Ctx()
	defer cancel()

	if !checkBotAdmin(c, chatID, func(msg string) { _, _ = m.ReplyText(c, msg, nil) }) {
		return false
	}

	// only admins + auth users can play if play mode is enabled
	if db.Instance.GetPlayMode(dbCtx, chatID) {
		admins, err := cache.GetAdmins(c, chatID, false)
		if err != nil {
			c.Logger.Warn("getAdmins error", "error", err)
			return false
		}

		senderID := m.SenderID()
		isAdmin := slices.ContainsFunc(admins, func(a *td.ChatMember) bool {
			return SenderID(a.MemberId) == senderID
		})

		if !isAdmin && !db.Instance.IsAuthUser(dbCtx, chatID, senderID) {
			_, _ = m.ReplyText(c, "🚫 Play mode is enabled. Only admins and authorized users can play.", nil)
			return false
		}
	}

	return true
}
