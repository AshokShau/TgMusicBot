/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 */

package handlers

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/vc"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/AshokShau/gotdbot"
)

// handleParticipant processes every chat-member status change for the bot and its assistant.
func handleParticipant(client *gotdbot.Client, update *gotdbot.UpdateChatMember) error {
	chatID := update.ChatId

	// Only handle group/channel updates.
	if chatID > 0 {
		return gotdbot.EndGroups
	}

	userID := senderID(update.NewChatMember.MemberId)

	assistant, _, err := vc.Calls.GetGroupAssistant(chatID)
	if err != nil {
		client.Logger.Error("Failed to get assistant for chat", "chat_id", chatID, "error", err)
		return gotdbot.EndGroups
	}

	assistantID := assistant.App.Me().ID

	// Only act when the bot itself or its assistant is involved.
	if !isBotOrAssistant(userID, client.Me.Id, assistantID) {
		return gotdbot.EndGroups
	}

	chat, err := getSupergroup(client, chatID)
	if err != nil {
		return gotdbot.EndGroups
	}

	if chat == nil {
		// getSupergroup already left the chat when appropriate.
		return gotdbot.EndGroups
	}

	if editableUsername(chat) != "" {
		vc.Calls.UpdateInviteLink(chatID, "https://t.me/"+editableUsername(chat))
	}

	go storeChatToDB(chatID)

	oldStatus := update.OldChatMember.Status
	newStatus := update.NewChatMember.Status

	if isAdmin(oldStatus) || isAdmin(newStatus) {
		cache.UpdateAdminCache(chatID, update.NewChatMember)
	}

	client.Logger.Debug("Member status changed",
		"user_id", userID,
		"old_status", oldStatus,
		"new_status", newStatus,
		"chat_id", chatID,
	)

	return dispatchStatusChange(client, chatID, userID, assistantID, oldStatus, newStatus, chat)
}

// dispatchStatusChange routes a member-status transition to the right handler.
func dispatchStatusChange(
	client *gotdbot.Client,
	chatID, userID, assistantID int64,
	oldStatus, newStatus gotdbot.ChatMemberStatus,
	chat *gotdbot.Supergroup,
) error {
	wasLeft := isStatus[*gotdbot.ChatMemberStatusLeft](oldStatus)
	nowLeft := isStatus[*gotdbot.ChatMemberStatusLeft](newStatus)
	wasMember := isStatus[*gotdbot.ChatMemberStatusMember](oldStatus)
	nowMember := isStatus[*gotdbot.ChatMemberStatusMember](newStatus)
	wasAdmin := isStatus[*gotdbot.ChatMemberStatusAdministrator](oldStatus)
	nowAdmin := isStatus[*gotdbot.ChatMemberStatusAdministrator](newStatus)
	nowBanned := isStatus[*gotdbot.ChatMemberStatusBanned](newStatus)
	wasBanned := isStatus[*gotdbot.ChatMemberStatusBanned](oldStatus)
	wasRestricted := isStatus[*gotdbot.ChatMemberStatusRestricted](oldStatus)
	nowRestricted := isStatus[*gotdbot.ChatMemberStatusRestricted](newStatus)

	switch {
	case wasLeft && (nowMember || nowAdmin || nowRestricted):
		return onJoin(client, chatID, userID, assistantID, chat)

	case (wasMember || wasAdmin || wasRestricted) && nowLeft:
		return onLeave(client, chatID, userID, assistantID)

	case nowBanned:
		return onBan(client, chatID, userID, assistantID)

	case wasBanned && nowLeft:
		return onUnban(chatID, userID)

	case nowRestricted || wasRestricted:
		return onRestriction(client, chatID, userID, assistantID, oldStatus, newStatus)

	default:
		return onPromotionOrDemotion(client, chatID, userID, wasAdmin, nowAdmin, chat)
	}
}

// onRestriction handles transitions involving chatMemberStatusRestricted.
func onRestriction(
	client *gotdbot.Client,
	chatID, userID, assistantID int64,
	oldStatus, newStatus gotdbot.ChatMemberStatus,
) error {
	_, wasRestricted := oldStatus.(*gotdbot.ChatMemberStatusRestricted)
	newRestricted, nowRestricted := newStatus.(*gotdbot.ChatMemberStatusRestricted)

	switch {
	case wasRestricted && !nowRestricted:
		client.Logger.Info("User restriction lifted", "user_id", userID, "chat_id", chatID)
		updateMembershipCache(chatID, userID, newStatus)

	case !wasRestricted && nowRestricted:
		client.Logger.Info("User restricted in chat", "user_id", userID, "chat_id", chatID)
		updateMembershipCache(chatID, userID, newStatus)

		if userID == assistantID {
			msg := fmt.Sprintf(
				"⚠️ My assistant has been restricted in this chat.\n\n"+
					"If this was a mistake, please unrestrict <code>%d</code>.",
				assistantID,
			)
			if _, err := client.SendTextMessage(chatID, msg, &gotdbot.SendTextMessageOpts{
				ParseMode: "HTML",
			}); err != nil {
				return err
			}
		}

	default:
		client.Logger.Info("User permissions updated while restricted",
			"user_id", userID,
			"chat_id", chatID,
			"can_send_basic_messages", newRestricted.Permissions.CanSendBasicMessages,
		)
		updateMembershipCache(chatID, userID, newStatus)
		//logPermissionDiff(client, chatID, userID, oldStatus.(*gotdbot.ChatMemberStatusRestricted).Permissions, newRestricted.Permissions)
	}

	return nil
}

func onJoin(
	client *gotdbot.Client,
	chatID, userID, assistantID int64,
	chat *gotdbot.Supergroup,
) error {
	client.Logger.Info("User joined chat", "user_id", userID, "chat_id", chatID)
	if userID == client.Me.Id {
		client.Logger.Info("Bot joined chat", "chat_id", chatID)
		sendJoinLog(client, chatID, chat)
	}

	updateMembershipCache(chatID, userID, &gotdbot.ChatMemberStatusMember{})
	return nil
}

func onLeave(client *gotdbot.Client, chatID, userID, assistantID int64) error {
	client.Logger.Info("User left chat", "user_id", userID, "chat_id", chatID)
	if userID == assistantID {
		cache.ChatCache.ClearChat(chatID)
	}

	updateMembershipCache(chatID, userID, &gotdbot.ChatMemberStatusLeft{})
	if userID == client.Me.Id {
		if err := vc.Calls.Stop(chatID, true); err != nil {
			client.Logger.Error("Failed to stop VC after leave", "error", err)
		}
	}

	return nil
}

func onBan(client *gotdbot.Client, chatID, userID, assistantID int64) error {
	client.Logger.Debug("User banned from chat", "user_id", userID, "chat_id", chatID)
	updateMembershipCache(chatID, userID, &gotdbot.ChatMemberStatusBanned{})

	if userID == assistantID {
		cache.ChatCache.ClearChat(chatID)
		msg := fmt.Sprintf(
			"🚫 My assistant has been banned from this chat.\n\n"+
				"If this was a mistake, please unban <code>%d</code>.",
			assistantID,
		)
		if _, err := client.SendTextMessage(chatID, msg, &gotdbot.SendTextMessageOpts{
			ParseMode: "HTML",
		}); err != nil {
			return err
		}
	}

	if userID == client.Me.Id {
		if err := vc.Calls.Stop(chatID, true); err != nil {
			client.Logger.Error("Failed to stop VC after ban", "error", err)
		}
	}

	return nil
}

func onUnban(chatID, userID int64) error {
	slog.Info("User unbanned from chat", "user_id", userID, "chat_id", chatID)
	updateMembershipCache(chatID, userID, &gotdbot.ChatMemberStatusLeft{})
	return nil
}

func onPromotionOrDemotion(
	client *gotdbot.Client,
	chatID, userID int64,
	wasAdmin, nowAdmin bool,
	chat *gotdbot.Supergroup,
) error {
	switch {
	case !wasAdmin && nowAdmin:
		client.Logger.Info("User promoted in chat", "user_id", userID, "chat_id", chatID)
		updateMembershipCacheByID(chatID, userID, &gotdbot.ChatMemberStatusAdministrator{})

	case wasAdmin && !nowAdmin:
		client.Logger.Info("User demoted in chat", "user_id", userID, "chat_id", chatID)
		updateMembershipCacheByID(chatID, userID, &gotdbot.ChatMemberStatusMember{})

	default:
		client.Logger.Info("onPromotionOrDemotio", "user_id", userID, "chat_id", chatID)
	}

	return nil
}

// getSupergroup fetches the Supergroup for a chat ID.
// Returns nil (and leaves the chat) when the chat should be abandoned.
func getSupergroup(client *gotdbot.Client, chatID int64) (*gotdbot.Supergroup, error) {
	rawID := stripChannelPrefix(chatID)
	chat, err := client.GetSupergroup(rawID)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid supergroup identifier") {
			_ = client.LeaveChat(chatID)
			return nil, nil
		}

		client.Logger.Error("Failed to fetch supergroup", "chat_id", chatID, "error", err)
		return nil, err
	}

	if chat.IsDirectMessagesGroup {
		_ = client.LeaveChat(chatID)
		return nil, nil
	}

	return chat, nil
}

// stripChannelPrefix converts a full channel ID (e.g. -1001234567890) to its
// bare supergroup ID (1234567890) as expected by GetSupergroup.
func stripChannelPrefix(chatID int64) int64 {
	s := strings.TrimPrefix(strconv.FormatInt(chatID, 10), "-100")
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

func editableUsername(chat *gotdbot.Supergroup) string {
	if chat.Usernames != nil {
		return chat.Usernames.EditableUsername
	}
	return ""
}

func sendJoinLog(client *gotdbot.Client, chatID int64, _ *gotdbot.Supergroup) {
	text := fmt.Sprintf("<b>🤖 Bot Joined a New Chat</b>\n📌 <b>Chat ID:</b> <code>%d</code>", chatID)
	if _, err := client.SendTextMessage(config.LoggerId, text, &gotdbot.SendTextMessageOpts{
		ParseMode: "HTML",
	}); err != nil {
		client.Logger.Warn("Failed to send join log", "error", err)
	}
}

// storeChatToDB persists the chat ID in the database; runs in a goroutine.
func storeChatToDB(chatID int64) {
	slog.Debug("Storing chat reference", "chat_id", chatID)
	if err := db.Instance.AddChat(chatID); err != nil {
		slog.Error("Failed to add chat to database", "chat_id", chatID, "error", err)
	}
}

// senderID extracts the numeric ID from a MessageSender union type.
func senderID(sender gotdbot.MessageSender) int64 {
	switch s := sender.(type) {
	case *gotdbot.MessageSenderUser:
		return s.UserId
	case *gotdbot.MessageSenderChat:
		return s.ChatId
	default:
		return 0
	}
}

// isBotOrAssistant reports whether the given user is the bot or its assistant.
func isBotOrAssistant(userID, botID, assistantID int64) bool {
	return userID == botID || userID == assistantID
}

// isAdmin reports whether a ChatMemberStatus is admin-level.
func isAdmin(status gotdbot.ChatMemberStatus) bool {
	switch status.(type) {
	case *gotdbot.ChatMemberStatusAdministrator, *gotdbot.ChatMemberStatusCreator:
		return true
	default:
		return false
	}
}

func isStatus[T gotdbot.ChatMemberStatus](status gotdbot.ChatMemberStatus) bool {
	_, ok := status.(T)
	return ok
}

// updateMembershipCache updates the VC membership for the assistant only.
func updateMembershipCache(chatID, userID int64, status gotdbot.ChatMemberStatus) {
	vc.Calls.UpdateMembership(chatID, userID, status)
}

// updateMembershipCacheByID is like updateMembershipCache but resolves the
func updateMembershipCacheByID(chatID, userID int64, status gotdbot.ChatMemberStatus) {
	vc.Calls.UpdateMembership(chatID, userID, status)
}
