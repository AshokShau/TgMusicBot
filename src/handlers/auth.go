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
	"errors"
	"fmt"

	"ashokshau/tgmusic/src/core/db"

	td "github.com/AshokShau/gotdbot"
)

// getTargetUserID gets the user ID from a message.
func getTargetUserID(c *td.Client, m *td.Message) (int64, error) {
	var userID int64
	if m.ReplyToMessageID() != 0 {
		replyMsg, err := m.GetRepliedMessage(c)
		if err != nil {
			return 0, err
		}
		userID = replyMsg.SenderID()
	}

	if userID == 0 {
		return 0, errors.New("no user specified")
	}

	if m.SenderID() == userID {
		return 0, errors.New("cannot perform action on yourself")
	}

	return userID, nil
}

// authListHandler handles the /auth command.
func authListHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}

	m := ctx.EffectiveMessage
	if m.IsPrivate() {
		return nil
	}

	chatID := m.ChatId
	ctx2, cancel := db.Ctx()
	defer cancel()

	authUser := db.Instance.GetAuthUsers(ctx2, chatID)
	if authUser == nil || len(authUser) == 0 {
		_, _ = m.ReplyText(c, "ℹ️ No authorized users.", nil)
		return nil
	}

	text := "<b>Authorized Users:</b>\n\n"
	for _, uid := range authUser {
		text += fmt.Sprintf("• <a href=\"tg://user?id=%d\">%d</a>\n", uid, uid)
	}

	_, _ = m.ReplyText(c, text, &td.SendTextMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}

// addAuthHandler handles the /addauth command.
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
		_, _ = m.ReplyText(c, "⚠️ Failed to get user admin status (cache or fetch failed).", nil)
		return td.EndGroups
	}

	switch UserStatus.Status.(type) {
	case *td.ChatMemberStatusCreator, *td.ChatMemberStatusAdministrator:
		// User is an admin, proceed
	default:
		_, _ = m.ReplyText(c, "❌ You must be an admin to use this command.", nil)
		return td.EndGroups
	}

	ctx2, cancel := db.Ctx()
	defer cancel()

	userID, err := getTargetUserID(c, m)
	if err != nil {
		_, _ = m.ReplyText(c, err.Error(), nil)
		return nil
	}

	if db.Instance.IsAuthUser(ctx2, chatID, userID) {
		_, _ = m.ReplyText(c, "User is already authorized.", nil)
		return nil
	}

	if err = db.Instance.AddAuthUser(ctx2, chatID, userID); err != nil {
		c.Logger.Error("Failed to add authorized user", "error", err)
		_, _ = m.ReplyText(c, "Error adding user.", nil)
		return nil
	}

	_, err = m.ReplyText(c, fmt.Sprintf("✅ User %d authorized.", userID), nil)
	return err
}

// removeAuthHandler handles the /removeauth command.
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
		_, _ = m.ReplyText(c, "⚠️ Failed to get user admin status (cache or fetch failed).", nil)
		return td.EndGroups
	}

	switch UserStatus.Status.(type) {
	case *td.ChatMemberStatusCreator, *td.ChatMemberStatusAdministrator:
		// User is an admin, proceed
	default:
		_, _ = m.ReplyText(c, "❌ You must be an admin to use this command.", nil)
		return td.EndGroups
	}

	ctx2, cancel := db.Ctx()
	defer cancel()

	userID, err := getTargetUserID(c, m)
	if err != nil {
		_, _ = m.ReplyText(c, err.Error(), nil)
		return nil
	}

	if !db.Instance.IsAuthUser(ctx2, chatID, userID) {
		_, _ = m.ReplyText(c, "User is not authorized.", nil)
		return nil
	}

	if err := db.Instance.RemoveAuthUser(ctx2, chatID, userID); err != nil {
		c.Logger.Error("Failed to remove authorized user", "error", err)
		_, _ = m.ReplyText(c, "Error removing user.", nil)
		return nil
	}

	_, err = m.ReplyText(c, fmt.Sprintf("✅ User %d removed from authorized list.", userID), nil)
	return err
}
