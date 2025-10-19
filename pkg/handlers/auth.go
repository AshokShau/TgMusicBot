package handlers

import (
	"errors"
	"fmt"

	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

// getTargetUserID gets the user ID from a message.
// It takes a telegram.NewMessage object as input.
// It returns the user ID and an error if any.
func getTargetUserID(m *telegram.NewMessage, langCode string) (int64, error) {
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
		ux, ok := user.(*telegram.UserObj)
		if !ok {
			return 0, errors.New(lang.GetString(langCode, "auth_user_not_found"))
		}
		userID = ux.ID
	}

	if userID == 0 {
		return 0, errors.New(lang.GetString(langCode, "auth_no_user_specified"))
	}

	if m.SenderID() == userID {
		return 0, errors.New(lang.GetString(langCode, "auth_action_on_self"))
	}

	return userID, nil
}

// authListHandler handles the /auth command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func authListHandler(m *telegram.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	authUser := db.Instance.GetAuthUsers(ctx, chatID)
	if authUser == nil || len(authUser) == 0 {
		_, _ = m.Reply(lang.GetString(langCode, "no_auth_users"))
		return nil
	}

	text := lang.GetString(langCode, "auth_users_list")
	for _, uid := range authUser {
		text += fmt.Sprintf("• <code>%d</code>\n", uid)
	}

	_, err := m.Reply(text)
	return err
}

// addAuthHandler handles the /addauth command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func addAuthHandler(m *telegram.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	userID, err := getTargetUserID(m, langCode)
	if err != nil {
		_, _ = m.Reply(err.Error())
		return nil
	}

	if db.Instance.IsAuthUser(ctx, chatID, userID) {
		_, _ = m.Reply(lang.GetString(langCode, "user_already_authed"))
		return nil
	}

	if err := db.Instance.AddAuthUser(ctx, chatID, userID); err != nil {
		gologging.Error("Failed to add authorized user:", err)
		_, _ = m.Reply(lang.GetString(langCode, "add_auth_error"))
		return nil
	}

	_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "user_authed"), userID))
	return err
}

// removeAuthHandler handles the /removeauth command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func removeAuthHandler(m *telegram.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	userID, err := getTargetUserID(m, langCode)
	if err != nil {
		_, _ = m.Reply(err.Error())
		return nil
	}

	if !db.Instance.IsAuthUser(ctx, chatID, userID) {
		_, _ = m.Reply(lang.GetString(langCode, "user_not_authed"))
		return nil
	}

	if err := db.Instance.RemoveAuthUser(ctx, chatID, userID); err != nil {
		gologging.Error("Failed to remove authorized user:", err)
		_, _ = m.Reply(lang.GetString(langCode, "remove_auth_error"))
		return nil
	}

	_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "user_unauthed"), userID))
	return err
}
