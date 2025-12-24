/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"errors"
	"fmt"

	"ashokshau/tgmusic/src/core/db"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// getTargetUserID gets the user ID from a message.
func getTargetUserID(m *tg.NewMessage) (int64, error) {
	var userID int64

	if m.IsReply() {
		replyMsg, err := m.GetReplyMessage()
		if err != nil {
			return 0, err
		}
		userID = replyMsg.SenderID()
	} else if len(m.Args()) > 0 {
		user, err := m.Client.ResolveUsername(m.Args())
		if err != nil {
			return 0, err
		}
		ux, ok := user.(*tg.UserObj)
		if !ok {
			return 0, errors.New("user not found")
		}
		userID = ux.ID
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
func authListHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()

	authUser := db.Instance.GetAuthUsers(ctx, chatID)
	if authUser == nil || len(authUser) == 0 {
		_, _ = m.Reply("ℹ️ No authorized users found.")
		return nil
	}

	text := "<b>🔐 Authorized Users:</b>\n\n"
	for _, uid := range authUser {
		text += fmt.Sprintf("• <code>%d</code>\n", uid)
	}

	_, err := m.Reply(text)
	return err
}

// addAuthHandler handles the /addauth command.
func addAuthHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()

	userID, err := getTargetUserID(m)
	if err != nil {
		_, _ = m.Reply(err.Error())
		return nil
	}

	if db.Instance.IsAuthUser(ctx, chatID, userID) {
		_, _ = m.Reply("User is already authorized.")
		return nil
	}

	if err := db.Instance.AddAuthUser(ctx, chatID, userID); err != nil {
		logger.Error("Failed to add authorized user:", err)
		_, _ = m.Reply("Something went wrong while adding the user.")
		return nil
	}

	_, err = m.Reply(fmt.Sprintf("✅ User (%d) has been successfully granted authorization permissions.", userID))
	return err
}

// removeAuthHandler handles the /removeauth command.
func removeAuthHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()

	userID, err := getTargetUserID(m)
	if err != nil {
		_, _ = m.Reply(err.Error())
		return nil
	}

	if !db.Instance.IsAuthUser(ctx, chatID, userID) {
		_, _ = m.Reply("User is not authorized.")
		return nil
	}

	if err := db.Instance.RemoveAuthUser(ctx, chatID, userID); err != nil {
		logger.Error("Failed to remove authorized user:", err)
		_, _ = m.Reply("Something went wrong while removing the user.")
		return nil
	}

	_, err = m.Reply(fmt.Sprintf("✅ User (%d) has been successfully removed from the authorized users list.", userID))
	return err
}
