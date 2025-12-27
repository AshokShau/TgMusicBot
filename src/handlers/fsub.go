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
	"sync"

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"

	"github.com/amarnathcjd/gogram/telegram"
)

// addFsubHandler handles the /addfsub command to set force subscribe channel/group.
// Usage: /addfsub <fsub_channel_id>
// This command only works in private chat with bot owner.
// Sets a GLOBAL fsub that applies to ALL groups.
func addFsubHandler(m *telegram.NewMessage) error {
	ctx, cancel := db.Ctx()
	defer cancel()

	// Must be in private chat
	if !m.IsPrivate() {
		_, _ = m.Reply("❌ This command only works in private chat.")
		return nil
	}

	args := m.Args()
	if args == "" {
		_, _ = m.Reply("❌ Usage: /addfsub <channel_id>\n\nExample:\n/addfsub -1001234567890")
		return nil
	}

	// Parse fsub channel ID
	fsubID, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		_, _ = m.Reply("❌ Invalid channel ID. Should be like -1001234567890")
		return nil
	}

	var fsubLink string
	var fsubTitle string

	// Try to get chat info
	chat, err := m.Client.GetChat(fsubID)
	if err != nil {
		fsubTitle = fmt.Sprintf("Channel %d", fsubID)
	} else if chat != nil {
		fsubTitle = chat.Title
	} else {
		fsubTitle = fmt.Sprintf("Channel %d", fsubID)
	}

	// Try to get invite link
	raw, err := m.Client.GetChatInviteLink(fsubID)
	if err == nil {
		if exported, ok := raw.(*telegram.ChatInviteExported); ok && exported.Link != "" {
			fsubLink = exported.Link
		}
	}

	// If still no link, try to create private channel link format
	if fsubLink == "" {
		// Convert channel ID to proper format for t.me/c/ link
		channelNum := fsubID
		if channelNum < 0 {
			// Remove -100 prefix
			channelStr := fmt.Sprintf("%d", -channelNum)
			if len(channelStr) > 3 && channelStr[:3] == "100" {
				channelStr = channelStr[3:]
			}
			fsubLink = fmt.Sprintf("https://t.me/c/%s/1", channelStr)
		} else {
			fsubLink = fmt.Sprintf("https://t.me/c/%d/1", channelNum)
		}
	}

	// Save to database with key 0 (global fsub)
	if err := db.Instance.SetFSub(ctx, 0, fsubID, fsubLink); err != nil {
		_, _ = m.Reply(fmt.Sprintf("❌ Error: %s", err.Error()))
		return nil
	}

	_, _ = m.Reply(fmt.Sprintf("✅ Force subscribe global berhasil diatur!\n\n� <b>Channel:</b> %s\n🆔 <b>ID:</b> <code>%d</code>\n🔗 <b>Link:</b> %s\n\n⚠️ Semua user di semua grup harus join channel ini untuk menggunakan /play", fsubTitle, fsubID, fsubLink))
	return nil
}

// removeFsubHandler handles the /rmfsub and /delfsub commands.
// Usage: /rmfsub <chat_id>
// This command only works in private chat with bot owner.
func removeFsubHandler(m *telegram.NewMessage) error {
	ctx, cancel := db.Ctx()
	defer cancel()

	// Must be in private chat
	if !m.IsPrivate() {
		_, _ = m.Reply("❌ This command only works in private chat.")
		return nil
	}

	// Check if global fsub is set (key 0)
	fsubID, _, _ := db.Instance.GetFSub(ctx, 0)
	if fsubID == 0 {
		_, _ = m.Reply("ℹ️ No global force subscribe is set.")
		return nil
	}

	// Remove from database
	if err := db.Instance.RemoveFSub(ctx, 0); err != nil {
		_, _ = m.Reply(fmt.Sprintf("❌ Error: %s", err.Error()))
		return nil
	}

	_, _ = m.Reply("✅ Global force subscribe has been removed.")
	return nil
}

// fsubStatusHandler handles the /fsub command to show current fsub status.
// This command only works in private chat with bot owner.
func fsubStatusHandler(m *telegram.NewMessage) error {
	ctx, cancel := db.Ctx()
	defer cancel()

	// Must be in private chat
	if !m.IsPrivate() {
		_, _ = m.Reply("❌ This command only works in private chat.")
		return nil
	}

	// Get global fsub (key 0)
	fsubID, fsubLink, _ := db.Instance.GetFSub(ctx, 0)
	if fsubID == 0 {
		_, _ = m.Reply("ℹ️ No global force subscribe is set.\n\nUse /addfsub <channel_id> to set one.")
		return nil
	}

	// Display fsub info
	fsubTitle := fsubLink
	if fsubLink != "" {
		fsubTitle = fmt.Sprintf("<a href='%s'>%s</a>", fsubLink, fsubLink)
	}

	_, _ = m.Reply(fmt.Sprintf("📋 <b>Global Force Subscribe</b>\n\n🔗 Channel: %s\n🆔 Channel ID: <code>%d</code>", fsubTitle, fsubID))
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
	storedUserID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil
	}

	// If storedUserID is 0, it means anyone can verify (anonymous/channel post)
	// Otherwise, only the specific user who triggered can verify
	if storedUserID != 0 && cb.SenderID != storedUserID {
		_, _ = cb.Answer("This button is not for you.", &telegram.CallbackOptions{Alert: true})
		return nil
	}

	langCode := db.Instance.GetLang(ctx, chatID)

	if action != "verify" {
		return nil
	}

	// Get GLOBAL fsub settings (key 0)
	fsubID, _, _ := db.Instance.GetFSub(ctx, 0)
	if fsubID == 0 {
		_, _ = cb.Answer(lang.GetString(langCode, "fsub_not_set"), &telegram.CallbackOptions{Alert: true})
		_, _ = cb.Delete()
		return nil
	}

	// Check if the person clicking the button is member of fsub
	// Use cb.SenderID (the actual person clicking) not storedUserID
	isMember := checkUserMembership(cb.Client, fsubID, cb.SenderID)
	if !isMember {
		_, _ = cb.Answer(lang.GetString(langCode, "fsub_not_member"), &telegram.CallbackOptions{Alert: true})
		return nil
	}

	// User is verified - check if there's a pending play request
	pendingPlay := GetPendingPlay(chatID)
	if pendingPlay != nil && pendingPlay.Message != nil {
		// Delete the verification message
		_, _ = cb.Delete()

		// Execute the pending play
		_, _ = cb.Answer(lang.GetString(langCode, "fsub_joined"), &telegram.CallbackOptions{Alert: true})

		// Call handlePlaySkipFsub to bypass fsub check (already verified)
		go handlePlaySkipFsub(pendingPlay.Message, pendingPlay.IsVideo)
		return nil
	}

	// No pending play, just show verified message
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

// PendingPlay stores a pending play request for anonymous verification
type PendingPlay struct {
	Query   string
	IsVideo bool
	Message *telegram.NewMessage
}

// pendingPlays stores pending play requests indexed by chatID
var pendingPlays = make(map[int64]*PendingPlay)
var pendingPlaysMux = sync.RWMutex{}

// StorePendingPlay stores a pending play request for later execution after verification
func StorePendingPlay(chatID int64, query string, isVideo bool, m *telegram.NewMessage) {
	pendingPlaysMux.Lock()
	defer pendingPlaysMux.Unlock()
	pendingPlays[chatID] = &PendingPlay{
		Query:   query,
		IsVideo: isVideo,
		Message: m,
	}
}

// GetPendingPlay retrieves and removes a pending play request
func GetPendingPlay(chatID int64) *PendingPlay {
	pendingPlaysMux.Lock()
	defer pendingPlaysMux.Unlock()
	if pp, ok := pendingPlays[chatID]; ok {
		delete(pendingPlays, chatID)
		return pp
	}
	return nil
}

// CheckFsubAndNotify checks fsub membership and sends notification if not joined.
// Returns true if user is allowed to proceed, false otherwise.
// For anonymous users, returns false and stores pending play for later execution.
func CheckFsubAndNotify(m *telegram.NewMessage, query string, isVideo bool) bool {
	chatID := m.ChannelID()
	userID := m.SenderID()

	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	// Get GLOBAL fsub settings (key 0)
	fsubID, fsubLink, _ := db.Instance.GetFSub(ctx, 0)
	if fsubID == 0 {
		// No global fsub set, allow
		return true
	}

	// For anonymous users (channel posts), we can't check membership directly
	// Store the pending play and ask for verification
	if userID == 0 || m.Sender == nil {
		// Store pending play for this chat
		StorePendingPlay(chatID, query, isVideo, m)

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

// CheckFsubOnly checks fsub membership without storing pending play (for non-play commands)
func CheckFsubOnly(m *telegram.NewMessage) bool {
	chatID := m.ChannelID()
	userID := m.SenderID()

	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	// Get GLOBAL fsub settings (key 0)
	fsubID, fsubLink, _ := db.Instance.GetFSub(ctx, 0)
	if fsubID == 0 {
		return true
	}

	// For anonymous users, allow
	if userID == 0 || m.Sender == nil {
		return true
	}

	// Check if user is member
	isMember := checkUserMembership(m.Client, fsubID, userID)
	if isMember {
		return true
	}

	// User is not a member
	fsubTitle := getFsubTitle(m.Client, fsubID, fsubLink)
	msg := fmt.Sprintf(lang.GetString(langCode, "fsub_not_joined"), fsubTitle)
	_, _ = m.Reply(msg, &telegram.SendOptions{
		ReplyMarkup: core.FsubKeyboard(fsubLink, chatID, userID),
	})

	return false
}

// getFsubTitle gets the title of fsub chat for display.
func getFsubTitle(client *telegram.Client, fsubID int64, fsubLink string) string {
	if fsubLink != "" {
		return fmt.Sprintf("<a href='%s'>%s</a>", fsubLink, fsubLink)
	}
	return fmt.Sprintf("<code>%d</code>", fsubID)
}
