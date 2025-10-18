package handlers

import (
	"fmt"
	"tgmusic/pkg/core/db"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

// getTargetUserID gets the user ID from a message.
// It takes a telegram.NewMessage object as input.
// It returns the user ID and an error if any.
func getTargetUserID(m *telegram.NewMessage) (int64, error) {
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
			return 0, fmt.Errorf("user not found")
		}
		userID = ux.ID
	}

	if userID == 0 {
		return 0, fmt.Errorf("no user specified")
	}

	if m.SenderID() == userID {
		return 0, fmt.Errorf("cannot perform action on yourself")
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
	chatId, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()

	authUser := db.Instance.GetAuthUsers(ctx, chatId)
	if authUser == nil || len(authUser) == 0 {
		_, _ = m.Reply("ℹ️ No authorized users found.")
		return nil
	}

	text := fmt.Sprintf("<b>🔐 Authorized Users:</b>\n\n")
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
		gologging.Error("Failed to add authorized user:", err)
		_, _ = m.Reply("Something went wrong while adding the user.")
		return nil
	}

	_, err = m.Reply(fmt.Sprintf("✅ User (%d) has been successfully granted authorization permissions.", userID))
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
		gologging.Error("Failed to remove authorized user:", err)
		_, _ = m.Reply("Something went wrong while removing the user.")
		return nil
	}

	_, err = m.Reply(fmt.Sprintf("✅ User (%d) has been successfully removed from the authorized users list.", userID))
	return err
}
