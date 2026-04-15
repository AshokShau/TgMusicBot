/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"fmt"

	td "github.com/AshokShau/gotdbot"
)

func authListHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}

	m := ctx.EffectiveMessage
	if m.IsPrivate() {
		return nil
	}

	chatID := m.ChatId

	authUser := db.Instance.GetAuthUsers(chatID)
	if authUser == nil || len(authUser) == 0 {
		_, _ = m.ReplyText(c, "No authorized users found.", nil)
		return nil
	}

	text := "<b>Authorized Users</b>\n\n"
	for _, uid := range authUser {
		text += fmt.Sprintf("• <a href=\"tg://user?id=%d\">%d</a>\n", uid, uid)
	}

	_, _ = m.ReplyText(c, text, replyOpts)
	return td.EndGroups
}

func addAuthHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}

	m := ctx.EffectiveMessage
	if m.IsPrivate() {
		return td.EndGroups
	}

	chatID := m.ChatId

	UserStatus, err := cache.GetUserAdmin(c, chatID, m.SenderID(), false)
	if err != nil {
		c.Logger.Warn("GetUserAdmin error", "error", err)
		_, _ = m.ReplyText(c, "Unable to verify administrator status.", nil)
		return td.EndGroups
	}

	switch UserStatus.Status.(type) {
	case *td.ChatMemberStatusCreator, *td.ChatMemberStatusAdministrator:
	default:
		_, _ = m.ReplyText(c, "You must be an administrator to use this command.", nil)
		return td.EndGroups
	}

	userID, err := getTargetUserID(c, m)
	if err != nil {
		_, _ = m.ReplyText(c, err.Error(), nil)
		return nil
	}

	if db.Instance.IsAuthUser(chatID, userID) {
		_, _ = m.ReplyText(c, "This user is already authorized.", nil)
		return nil
	}

	if err = db.Instance.AddAuthUser(chatID, userID); err != nil {
		c.Logger.Error("Failed to add authorized user", "error", err)
		_, _ = m.ReplyText(c, "Failed to authorize the user.", nil)
		return nil
	}

	_, err = m.ReplyText(c, fmt.Sprintf("User %d has been authorized.", userID), nil)
	return err
}

func removeAuthHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}

	m := ctx.EffectiveMessage
	if m.IsPrivate() {
		return td.EndGroups
	}

	chatID := m.ChatId

	UserStatus, err := cache.GetUserAdmin(c, chatID, m.SenderID(), false)
	if err != nil {
		c.Logger.Warn("GetUserAdmin error", "error", err)
		_, _ = m.ReplyText(c, "Unable to verify administrator status.", nil)
		return td.EndGroups
	}

	switch UserStatus.Status.(type) {
	case *td.ChatMemberStatusCreator, *td.ChatMemberStatusAdministrator:
	default:
		_, _ = m.ReplyText(c, "You must be an administrator to use this command.", nil)
		return td.EndGroups
	}

	userID, err := getTargetUserID(c, m)
	if err != nil {
		_, _ = m.ReplyText(c, err.Error(), nil)
		return nil
	}

	if !db.Instance.IsAuthUser(chatID, userID) {
		_, _ = m.ReplyText(c, "This user is not authorized.", nil)
		return nil
	}

	if err := db.Instance.RemoveAuthUser(chatID, userID); err != nil {
		c.Logger.Error("Failed to remove authorized user", "error", err)
		_, _ = m.ReplyText(c, "Failed to remove authorized user.", nil)
		return nil
	}

	_, err = m.ReplyText(c, fmt.Sprintf("User %d has been removed from the authorized list.", userID), nil)
	return err
}
