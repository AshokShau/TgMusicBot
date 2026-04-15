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

func checkBotAdmin(c *td.Client, chatID int64, replyErr func(msg string)) bool {
	botStatus, err := cache.GetUserAdmin(c, chatID, c.Me.Id, false)
	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			replyErr("Bot is not an administrator in this chat. Please promote the bot with invite users permission.")
		} else {
			c.Logger.Warn("GetUserAdmin error", "error", err)
			replyErr("Unable to verify bot administrator status.")
		}
		return false
	}

	switch s := botStatus.Status.(type) {
	case *td.ChatMemberStatusCreator:
		return true
	case *td.ChatMemberStatusAdministrator:
		if s.Rights == nil || !s.Rights.CanInviteUsers {
			replyErr("The bot does not have permission to invite users.")
			return false
		}
		return true
	default:
		replyErr("Bot is not an administrator in this chat. Use /reload to refresh admin cache.")
		return false
	}
}

func adminMode(c *td.Client, ctx *td.Context) bool {
	m := ctx.EffectiveMessage
	if m.IsPrivate() {
		return false
	}

	chatID := m.ChatId

	if !checkBotAdmin(c, chatID, func(msg string) { _, _ = m.ReplyText(c, msg, nil) }) {
		return false
	}

	userID := m.SenderID()
	switch db.Instance.GetAdminMode(chatID) {
	case utils.Everyone:
		return true
	case utils.Admins:
		if db.Instance.IsAdmin(chatID, userID) || db.Instance.IsAuthUser(chatID, userID) {
			return true
		}
		_, _ = m.ReplyText(c, "You must be an administrator to use this command.", nil)
		return false
	default:
		_, _ = m.ReplyText(c, "You are not authorized to use this command.", nil)
		return false
	}
}

func adminModeCB(c *td.Client, cb *td.UpdateNewCallbackQuery) bool {
	chatID := cb.ChatId

	if !checkBotAdmin(c, chatID, func(msg string) { _ = cb.Answer(c, 0, true, msg, "") }) {
		return false
	}

	userID := cb.SenderUserId
	switch db.Instance.GetAdminMode(chatID) {
	case utils.Everyone:
		return true
	case utils.Admins:
		if db.Instance.IsAdmin(chatID, userID) || db.Instance.IsAuthUser(chatID, userID) {
			return true
		}
		_ = cb.Answer(c, 0, true, "You must be an administrator to use this action.", "")
		return false
	default:
		_ = cb.Answer(c, 0, true, "You are not authorized to use this action.", "")
		return false
	}
}

func playMode(c *td.Client, ctx *td.Context) bool {
	m := ctx.EffectiveMessage
	if m.IsPrivate() {
		return false
	}

	chatID := m.ChatID()

	if !checkBotAdmin(c, chatID, func(msg string) { _, _ = m.ReplyText(c, msg, nil) }) {
		return false
	}

	if db.Instance.GetPlayMode(chatID) {
		admins, err := cache.GetAdmins(c, chatID, false)
		if err != nil {
			c.Logger.Warn("getAdmins error", "error", err)
			return false
		}

		senderID := m.SenderID()
		isAdmin := slices.ContainsFunc(admins, func(a *td.ChatMember) bool {
			return SenderID(a.MemberId) == senderID
		})

		if !isAdmin && !db.Instance.IsAuthUser(chatID, senderID) {
			_, _ = m.ReplyText(c, "Play mode is enabled. Only administrators and authorized users can start playback.", nil)
			return false
		}
	}

	return true
}
