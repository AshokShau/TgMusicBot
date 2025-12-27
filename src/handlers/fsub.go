/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"

	"github.com/amarnathcjd/gogram/telegram"
)

// addFsubHandler handles the /addfsub command to set force subscribe channel/group.
func addFsubHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	args := m.Args()
	if args == "" {
		_, _ = m.Reply(lang.GetString(langCode, "fsub_usage"))
		return nil
	}

	// Parse the fsub target (can be @username or chat ID)
	var fsubID int64
	var fsubLink string
	var fsubTitle string

	// Try to resolve as username or chat ID
	if strings.HasPrefix(args, "@") {
		// It's a username
		peer, err := m.Client.ResolveUsername(strings.TrimPrefix(args, "@"))
		if err != nil {
			_, _ = m.Reply(lang.GetString(langCode, "fsub_invalid_chat"))
			return nil
		}

		switch p := peer.(type) {
		case *telegram.Channel:
			fsubID = p.ID
			fsubTitle = p.Title
			if p.Username != "" {
				fsubLink = fmt.Sprintf("https://t.me/%s", p.Username)
			}
		default:
			_, _ = m.Reply(lang.GetString(langCode, "fsub_invalid_chat"))
			return nil
		}
	} else {
		// Try to parse as chat ID
		id, err := strconv.ParseInt(args, 10, 64)
		if err != nil {
			_, _ = m.Reply(lang.GetString(langCode, "fsub_invalid_chat"))
			return nil
		}
		fsubID = id

		// Try to get chat info
		peer, err := m.Client.ResolveUsername(args)
		if err != nil {
			// Try to get info using GetChat
			chat, err := m.Client.GetChat(id)
			if err != nil {
				_, _ = m.Reply(lang.GetString(langCode, "fsub_invalid_chat"))
				return nil
			}

			// GetChat returns *ChatObj which has Title field
			if chat != nil {
				fsubTitle = chat.Title
			} else {
				fsubTitle = fmt.Sprintf("Chat %d", id)
			}
		} else {
			switch c := peer.(type) {
			case *telegram.Channel:
				fsubTitle = c.Title
				if c.Username != "" {
					fsubLink = fmt.Sprintf("https://t.me/%s", c.Username)
				}
			default:
				_, _ = m.Reply(lang.GetString(langCode, "fsub_only_channel_group"))
				return nil
			}
		}
	}

	// If no public link, try to get invite link
	if fsubLink == "" {
		raw, err := m.Client.GetChatInviteLink(fsubID)
		if err == nil {
			if exported, ok := raw.(*telegram.ChatInviteExported); ok && exported.Link != "" {
				fsubLink = exported.Link
			}
		}
	}

	// If still no link, use a placeholder
	if fsubLink == "" {
		fsubLink = fmt.Sprintf("https://t.me/c/%d", fsubID)
	}

	// Save to database
	if err := db.Instance.SetFSub(ctx, chatID, fsubID, fsubLink); err != nil {
		_, _ = m.Reply(fmt.Sprintf("❌ Error: %s", err.Error()))
		return nil
	}

	_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "fsub_set_success"), fsubTitle))
	return nil
}

// removeFsubHandler handles the /rmfsub and /delfsub commands.
func removeFsubHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	// Check if fsub is set
	fsubID, _, _ := db.Instance.GetFSub(ctx, chatID)
	if fsubID == 0 {
		_, _ = m.Reply(lang.GetString(langCode, "fsub_not_set"))
		return nil
	}

	// Remove from database
	if err := db.Instance.RemoveFSub(ctx, chatID); err != nil {
		_, _ = m.Reply(fmt.Sprintf("❌ Error: %s", err.Error()))
		return nil
	}

	_, _ = m.Reply(lang.GetString(langCode, "fsub_removed"))
	return nil
}

// fsubStatusHandler handles the /fsub command to show current fsub status.
func fsubStatusHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	fsubID, fsubLink, _ := db.Instance.GetFSub(ctx, chatID)
	if fsubID == 0 {
		_, _ = m.Reply(lang.GetString(langCode, "fsub_not_set"))
		return nil
	}

	// Try to get chat title
	fsubTitle := fmt.Sprintf("<code>%d</code>", fsubID)
	peer, err := m.Client.ResolveUsername("")
	if err == nil {
		if c, ok := peer.(*telegram.Channel); ok {
			if fsubLink != "" {
				fsubTitle = fmt.Sprintf("<a href='%s'>%s</a>", fsubLink, c.Title)
			} else {
				fsubTitle = c.Title
			}
		}
	} else {
		// Use the link as title
		if fsubLink != "" {
			fsubTitle = fmt.Sprintf("<a href='%s'>%s</a>", fsubLink, fsubLink)
		}
	}

	_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "fsub_current"), fsubTitle))
	return nil
}

// fsubCallbackHandler handles fsub verification callbacks.
func fsubCallbackHandler(cb *telegram.CallbackQuery) error {
	data := cb.DataString()
	ctx, cancel := db.Ctx()
	defer cancel()

	// Parse callback data: fsub_verify_{chatID}_{userID}
	parts := strings.Split(data, "_")
	if len(parts) < 4 {
		return nil
	}

	action := parts[1]
	chatID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil
	}
	userID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil
	}

	// Only the user who triggered can verify
	if cb.SenderID != userID {
		_, _ = cb.Answer("This button is not for you.", &telegram.CallbackOptions{Alert: true})
		return nil
	}

	langCode := db.Instance.GetLang(ctx, chatID)

	if action != "verify" {
		return nil
	}

	// Get fsub settings
	fsubID, _, _ := db.Instance.GetFSub(ctx, chatID)
	if fsubID == 0 {
		_, _ = cb.Answer(lang.GetString(langCode, "fsub_not_set"), &telegram.CallbackOptions{Alert: true})
		_, _ = cb.Delete()
		return nil
	}

	// Check if user is member of fsub
	isMember := checkUserMembership(cb.Client, fsubID, userID)
	if !isMember {
		_, _ = cb.Answer(lang.GetString(langCode, "fsub_not_member"), &telegram.CallbackOptions{Alert: true})
		return nil
	}

	// User is verified
	_, _ = cb.Answer(lang.GetString(langCode, "fsub_joined"), &telegram.CallbackOptions{Alert: true})
	_, _ = cb.Edit(lang.GetString(langCode, "fsub_joined"), &telegram.SendOptions{ReplyMarkup: core.CloseKeyboard()})

	return nil
}

// checkUserMembership checks if a user is a member of a chat/channel.
func checkUserMembership(client *telegram.Client, chatID, userID int64) bool {
	member, err := client.GetChatMember(chatID, userID)
	if err != nil {
		// User is not a member or error occurred
		return false
	}

	// Check participant status - member status indicates they are in the chat
	switch member.Status {
	case telegram.Member, telegram.Admin, telegram.Creator:
		return true
	default:
		return false
	}
}

// CheckFsubAndNotify checks fsub membership and sends notification if not joined.
// Returns true if user is allowed to proceed, false otherwise.
func CheckFsubAndNotify(m *telegram.NewMessage) bool {
	chatID := m.ChannelID()
	userID := m.SenderID()

	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	// Get fsub settings
	fsubID, fsubLink, _ := db.Instance.GetFSub(ctx, chatID)
	if fsubID == 0 {
		// No fsub set, allow
		return true
	}

	// For anonymous users (channel posts), we can't check membership
	// So we need to send a verification button
	if userID == 0 || m.Sender == nil {
		// Send verification message
		fsubTitle := getFsubTitle(m.Client, fsubID, fsubLink)
		msg := fmt.Sprintf(lang.GetString(langCode, "fsub_not_joined"), fsubTitle)
		_, _ = m.Reply(msg, &telegram.SendOptions{
			ReplyMarkup: core.FsubKeyboard(fsubLink, chatID, 0),
		})
		return false
	}

	// Check if user is member
	isMember := checkUserMembership(m.Client, fsubID, userID)
	if isMember {
		return true
	}

	// User is not a member, send join message
	fsubTitle := getFsubTitle(m.Client, fsubID, fsubLink)
	msg := fmt.Sprintf(lang.GetString(langCode, "fsub_not_joined"), fsubTitle)
	_, _ = m.Reply(msg, &telegram.SendOptions{
		ReplyMarkup: core.FsubKeyboard(fsubLink, chatID, userID),
	})

	return false
}

// getFsubTitle gets the title of fsub chat for display.
func getFsubTitle(client *telegram.Client, fsubID int64, fsubLink string) string {
	peer, err := client.ResolveUsername("")
	if err != nil {
		return fsubLink
	}

	if c, ok := peer.(*telegram.Channel); ok {
		if fsubLink != "" {
			return fmt.Sprintf("<a href='%s'>%s</a>", fsubLink, c.Title)
		}
		return c.Title
	}

	return fsubLink
}
