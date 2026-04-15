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

func handleParticipant(client *gotdbot.Client, ctx *gotdbot.Context) error {
	update := ctx.Update.UpdateChatMember
	if update == nil {
		return gotdbot.EndGroups
	}

	chatID := ctx.EffectiveChatId
	me := client.Me

	getMemberInfo := func(memberId gotdbot.MessageSender) (int64, string) {
		switch sender := memberId.(type) {

		case *gotdbot.MessageSenderUser:
			return sender.UserId,
				fmt.Sprintf(
					"User <a href=\"tg://user?id=%d\">%d</a>",
					sender.UserId,
					sender.UserId,
				)

		case *gotdbot.MessageSenderChat:
			return sender.ChatId,
				fmt.Sprintf("Chat %d", sender.ChatId)

		default:
			return 0, "Unknown"
		}
	}

	userID, _ := getMemberInfo(update.NewChatMember.MemberId)
	call, _, err := vc.Calls.GetGroupAssistant(chatID)
	if err != nil {
		client.Logger.Error("Failed to get assistant for chat", "chat_id", chatID, "error", err)
		return gotdbot.EndGroups
	}

	ubID := call.App.Me().ID

	if !isRelevantUser(userID, me.Id, ubID) {
		return gotdbot.EndGroups
	}

	s := strings.TrimPrefix(strconv.FormatInt(chatID, 10), "-100")
	rawChatId, _ := strconv.ParseInt(s, 10, 64)

	chat, err := client.GetSupergroup(rawChatId)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid supergroup identifier") {
			_ = client.LeaveChat(chatID)
			return gotdbot.EndGroups
		}

		client.Logger.Error("Failed to get chat", "chat_id", chatID, "error", err)
		return gotdbot.EndGroups
	}

	if chat.IsDirectMessagesGroup {
		_ = client.LeaveChat(chatID)
		return gotdbot.EndGroups
	}

	if chat.Usernames != nil && chat.Usernames.EditableUsername != "" {
		inviteLink := fmt.Sprintf("https://t.me/%s", chat.Usernames.EditableUsername)
		vc.Calls.UpdateInviteLink(chatID, inviteLink)
	}

	go storeChatReference(chatID)

	oldStatus := update.OldChatMember.Status
	newStatus := update.NewChatMember.Status

	if isAdmin(oldStatus) || isAdmin(newStatus) {
		cache.UpdateAdminCache(chatID, update.NewChatMember)
	}

	client.Logger.Debug("Status change: UserID= Old= New= ChatID=", "user_id", userID, "arg2", oldStatus, "arg3", newStatus, "chat_id", chatID)

	return handleParticipantStatusChange(
		client,
		chatID,
		userID,
		ubID,
		oldStatus,
		newStatus,
		chat,
	)
}

func storeChatReference(chatID int64) {

	slog.Debug("Storing chat reference for chat", "chat_id", chatID)

	if err := db.Instance.AddChat(chatID); err != nil {
		slog.Error("Failed to add chat  to database", "chat_id", chatID, "error", err)
	}
}

func isRelevantUser(userID, botID, assistantID int64) bool {
	return userID == botID || userID == assistantID
}

func isAdmin(status gotdbot.ChatMemberStatus) bool {
	switch status.(type) {
	case *gotdbot.ChatMemberStatusAdministrator, *gotdbot.ChatMemberStatusCreator:
		return true
	default:
		return false
	}
}

func handleParticipantStatusChange(
	client *gotdbot.Client,
	chatID int64,
	userID int64,
	ubID int64,
	oldStatus gotdbot.ChatMemberStatus,
	newStatus gotdbot.ChatMemberStatus,
	chat *gotdbot.Supergroup,
) error {

	_, oldLeft := oldStatus.(*gotdbot.ChatMemberStatusLeft)
	_, newLeft := newStatus.(*gotdbot.ChatMemberStatusLeft)

	_, oldMember := oldStatus.(*gotdbot.ChatMemberStatusMember)
	_, newMember := newStatus.(*gotdbot.ChatMemberStatusMember)

	_, oldAdmin := oldStatus.(*gotdbot.ChatMemberStatusAdministrator)
	_, newAdmin := newStatus.(*gotdbot.ChatMemberStatusAdministrator)

	_, newBanned := newStatus.(*gotdbot.ChatMemberStatusBanned)
	_, oldBanned := oldStatus.(*gotdbot.ChatMemberStatusBanned)

	switch {

	case oldLeft && (newMember || newAdmin):
		return handleJoin(client, chatID, userID, ubID, chat)

	case (oldMember || oldAdmin) && newLeft:
		return handleLeave(client, chatID, userID, ubID)

	case newBanned:
		return handleBan(client, chatID, userID, ubID)

	case oldBanned && newLeft:
		return handleUnban(chatID, userID)

	default:
		return handlePromotionDemotion(
			client,
			chatID,
			userID,
			oldAdmin,
			newAdmin,
			chat,
		)
	}
}

func handleJoin(
	client *gotdbot.Client,
	chatID int64,
	userID int64,
	ubID int64,
	chat *gotdbot.Supergroup,
) error {

	client.Logger.Info("User  joined chat", "user_id", userID, "chat_id", chatID)

	if userID == client.Me.Id {
		client.Logger.Info("Bot joined chat", "chat_id", chatID)
		sendJoinLog(client, chatID, chat)
	}

	updateStatusCache(chatID, userID, &gotdbot.ChatMemberStatusMember{})

	return nil
}

func sendJoinLog(client *gotdbot.Client, chatID int64, chat *gotdbot.Supergroup) {

	text := fmt.Sprintf(
		"<b>🤖 Bot Joined a New Chat</b>\n"+
			"📌 <b>Chat ID:</b> <code>%d</code>\n",
		chatID,
	)

	_, err := client.SendTextMessage(
		config.Conf.LoggerId,
		text,
		&gotdbot.SendTextMessageOpts{
			ParseMode: "HTML",
		},
	)

	if err != nil {
		client.Logger.Warn("Failed to send join log", "error", err)
	}
}

func handleLeave(client *gotdbot.Client, chatID, userID, ubID int64) error {
	client.Logger.Info("User  left chat", "user_id", userID, "chat_id", chatID)

	if userID == ubID {
		cache.ChatCache.ClearChat(chatID)
	}

	if userID == client.Me.Id {
		if err := vc.Calls.Stop(chatID); err != nil {
			client.Logger.Error("Failed to stop VC", "error", err)
		}
	}

	updateStatusCache(chatID, userID, &gotdbot.ChatMemberStatusLeft{})

	return nil
}

func handleBan(client *gotdbot.Client, chatID, userID, ubID int64) error {
	client.Logger.Debug("User  banned from chat", "user_id", userID, "chat_id", chatID)

	if userID == ubID {

		cache.ChatCache.ClearChat(chatID)

		message := fmt.Sprintf(
			"🚫 My assistant has been banned from this chat.\n\n"+
				"If this was a mistake please unban <code>%d</code>.",
			ubID,
		)

		_, err := client.SendTextMessage(
			chatID,
			message,
			&gotdbot.SendTextMessageOpts{
				ParseMode: "HTML",
			},
		)

		if err != nil {
			return err
		}
	}

	if userID == client.Me.Id {
		if err := vc.Calls.Stop(chatID); err != nil {
			client.Logger.Error("Failed stopping VC after ban", "error", err)
		}
	}

	updateStatusCache(chatID, userID, &gotdbot.ChatMemberStatusBanned{})

	return nil
}

func handleUnban(chatID, userID int64) error {
	slog.Info("User  unbanned from chat", "user_id", userID, "chat_id", chatID)
	updateStatusCache(chatID, userID, &gotdbot.ChatMemberStatusLeft{})

	return nil
}

func handlePromotionDemotion(
	client *gotdbot.Client,
	chatID int64,
	userID int64,
	oldAdmin bool,
	newAdmin bool,
	chat *gotdbot.Supergroup,
) error {

	isPromoted := !oldAdmin && newAdmin
	isDemoted := oldAdmin && !newAdmin

	if !isPromoted && !isDemoted {
		return nil
	}

	if isPromoted {
		client.Logger.Info("User  promoted in chat", "user_id", userID, "chat_id", chatID)
		updateStatusCache(chatID, userID, &gotdbot.ChatMemberStatusAdministrator{})
		return nil
	}

	client.Logger.Info("User  demoted in chat", "user_id", userID, "chat_id", chatID)
	updateStatusCache(chatID, userID, &gotdbot.ChatMemberStatusMember{})

	return nil
}

func updateStatusCache(chatID, userID int64, status gotdbot.ChatMemberStatus) {
	call, _, err := vc.Calls.GetGroupAssistant(chatID)
	if err != nil {
		return
	}

	ubID := call.App.Me().ID

	if userID == ubID {
		vc.Calls.UpdateMembership(chatID, userID, status)
	}
}
