/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/config"
	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"
	"ashokshau/tgmusic/src/vc"
	"fmt"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func handleVoiceChatMessage(m *telegram.NewMessage) error {
	if m.Action == nil {
		return nil
	}

	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()

	langCode := "en"
	if db.Instance != nil {
		langCode = db.Instance.GetLang(ctx, chatID)
	}

	action, ok := m.Action.(*telegram.MessageActionGroupCall)
	if !ok {
		return telegram.EndGroup
	}

	var message string

	if action.Duration == 0 {
		cache.ChatCache.ClearChat(chatID)
		message = lang.GetString(langCode, "watcher_vc_started")
	} else {
		cache.ChatCache.ClearChat(chatID)
		logger.Info("Voice chat ended. Duration: %d seconds", action.Duration)
		message = lang.GetString(langCode, "watcher_vc_ended")
	}

	if message != "" {
		_, _ = m.Client.SendMessage(chatID, message)
	}

	return telegram.EndGroup
}

// handleParticipant handles participant updates.
func handleParticipant(pu *telegram.ParticipantUpdate) error {
	if pu == nil || pu.Channel == nil {
		logger.Error("[handleParticipant] nil participant update or nil channel")
		return nil
	}

	client := pu.Client
	chatID := pu.ChannelID()
	userID := pu.UserID()
	chat := pu.Channel

	ctx, cancel := db.Ctx()
	defer cancel()

	langCode := db.Instance.GetLang(ctx, chatID)

	// Chat is not a Supergroup
	if chatID > 0 {
		text := fmt.Sprintf(
			lang.GetString(langCode, "watcher_not_supergroup"),
			chatID,
		)

		_, _ = client.SendMessage(chatID, text, &telegram.SendOptions{
			ReplyMarkup: core.AddMeMarkup(client.Me().Username),
			LinkPreview: false,
		})

		time.Sleep(1 * time.Second)
		_ = client.LeaveChannel(chatID)
		return nil
	}

	// Store chat reference in DB
	go func(chatID int64) {
		ctx, cancel := db.Ctx()
		defer cancel()
		_ = db.Instance.AddChat(ctx, chatID)
	}(chatID)

	if chat.Username != "" {
		vc.Calls.UpdateInviteLink(chatID, "https://t.me/"+chat.Username)
	}

	logger.Debug(
		"[handleParticipant] Update: Old=%T New=%T ChatID=%d UserID=%d",
		pu.Old, pu.New, chatID, userID,
	)

	oldStatus := getStatusFromParticipant(pu.Old)
	newStatus := getStatusFromParticipant(pu.New)

	logger.Debug(
		"[handleParticipant] old=%s new=%s chat=%d user=%d",
		oldStatus, newStatus, chatID, userID,
	)

	call, err := vc.Calls.GetGroupAssistant(chatID)
	if err != nil {
		logger.Error("[handleParticipant] get group assistant failed: %v", err)
		return nil
	}

	ubID := call.App.Me().ID

	// Only handle bot/self changes
	if userID != ubID && userID != client.Me().ID {
		logger.Debug("[handleParticipant] ignoring update for %d", userID)
		return nil
	}

	return handleParticipantStatusChange(client, chatID, userID, ubID, oldStatus, newStatus, chat)
}

// handleParticipantStatusChange handles participant status changes.
func handleParticipantStatusChange(
	client *telegram.Client,
	chatID int64,
	userID, ubID int64,
	oldStatus, newStatus string,
	channel *telegram.Channel,
) error {

	switch {
	case oldStatus == telegram.Left && (newStatus == telegram.Member || newStatus == telegram.Admin):
		return handleJoin(client, chatID, userID, ubID, channel)

	case (oldStatus == telegram.Member || oldStatus == telegram.Admin) && newStatus == telegram.Left:
		return handleLeaveOrKick(client, chatID, userID, ubID)

	case newStatus == telegram.Kicked:
		return handleBan(client, chatID, userID, ubID)

	case oldStatus == telegram.Kicked && newStatus == telegram.Left:
		return handleUnban(chatID, userID)

	default:
		return handlePromotionDemotion(client, chatID, userID, oldStatus, newStatus, channel)
	}
}

// handleJoin handles join events.
func handleJoin(client *telegram.Client, chatID, userID, ubID int64, channel *telegram.Channel) error {
	if userID == client.Me().ID {
		logger.Info("Bot joined chat %d. Initializing…", chatID)

		text := fmt.Sprintf(
			"<b>🤖 Bot Joined a New Chat</b>\n"+
				"📌 <b>Chat ID:</b> <code>%d</code>\n"+
				"🏷️ <b>Title:</b> %s\n"+
				"👥 <b>Type:</b> %s\n"+
				"👤 <b>Username:</b> @%s\n",
			chatID,
			channel.Title,
			getChatType(channel),
			channel.Username,
		)

		_, err := client.SendMessage(config.Conf.LoggerId, text, &telegram.SendOptions{LinkPreview: false})
		if err != nil {
			logger.Warn("Failed to send join log: %v", err)
		}
	}

	if userID == ubID {
		logger.Info("Assistant joined chat %d. Initializing…", chatID)
	}

	logger.Debug("User %d joined chat %d", userID, chatID)
	updateUbStatusCache(chatID, userID, telegram.Member)
	return nil
}

// handleLeaveOrKick handles leave or kick events.
func handleLeaveOrKick(client *telegram.Client, chatID, userID, ubId int64) error {
	logger.Debug("User %d left/kicked in %d", userID, chatID)

	if userID == ubId {
		logger.Info("Assistant left %d. Clearing call/cache…", chatID)
		cache.ChatCache.ClearChat(chatID)
	}

	if userID == client.Me().ID {
		logger.Info("Bot left chat %d. Stopping call…", chatID)
		_ = vc.Calls.Stop(chatID)
	}

	updateUbStatusCache(chatID, userID, telegram.Left)
	return nil
}

// handleBan handles ban events.
func handleBan(client *telegram.Client, chatID, userID, ubId int64) error {
	logger.Debug("User %d was banned in %d", userID, chatID)

	ctx, cancel := db.Ctx()
	defer cancel()

	langCode := db.Instance.GetLang(ctx, chatID)

	if userID == ubId {
		logger.Info("Assistant banned from chat %d. Cleaning up…", chatID)

		cache.ChatCache.ClearChat(chatID)

		_, err := client.SendMessage(chatID,
			fmt.Sprintf(lang.GetString(langCode, "watcher_assistant_banned"), ubId),
		)
		if err != nil {
			logger.Error("Failed to send assistant ban message: %v", err)
			return err
		}
	}

	if userID == client.Me().ID {
		logger.Info("Bot was banned in chat %d. Stopping call…", chatID)
		_ = vc.Calls.Stop(chatID)
	}

	updateUbStatusCache(chatID, userID, telegram.Kicked)
	return nil
}

// handleUnban handles unban events.
func handleUnban(chatID, userID int64) error {
	logger.Debug("User %d was unbanned in chat %d", userID, chatID)
	updateUbStatusCache(chatID, userID, telegram.Left)
	return nil
}

// handlePromotionDemotion handles promotion/demotion events.
func handlePromotionDemotion(
	client *telegram.Client,
	chatID, userID int64,
	oldStatus, newStatus string,
	channel *telegram.Channel,
) error {

	isPromoted := oldStatus != telegram.Admin && newStatus == telegram.Admin
	isDemoted := oldStatus == telegram.Admin && newStatus != telegram.Admin

	if !isPromoted && !isDemoted {
		return nil
	}

	action := "promoted"
	if isDemoted {
		action = "demoted"
	}

	if userID == client.Me().ID {
		if isPromoted {
			logger.Info("Bot promoted in %d → refreshing admin cache", chatID)
			_, _ = cache.GetAdmins(client, chatID, true)
		} else {
			logger.Info("Bot demoted in %d → clearing admin cache", chatID)
			cache.ClearAdminCache(chatID)
		}

		_ = sendAdminStatusLog(client, chatID, userID, action, channel)
	} else {
		logger.Debug("User %d was %s in chat %d", userID, action, chatID)
	}

	vc.Calls.UpdateMembership(chatID, userID, newStatus)
	return nil
}

// sendAdminStatusLog sends admin status change logs.
func sendAdminStatusLog(client *telegram.Client, chatID, userID int64, action string, ch *telegram.Channel) error {
	text := fmt.Sprintf(
		"<b>⚠️ Admin Status Changed</b>\n"+
			"📌 <b>Chat:</b> %s (<code>%d</code>)\n"+
			"👤 <b>User:</b> <code>%d</code>\n"+
			"🔧 <b>Action:</b> %s\n",
		ch.Title,
		chatID,
		userID,
		strings.Title(action),
	)

	_, err := client.SendMessage(config.Conf.LoggerId, text, &telegram.SendOptions{LinkPreview: false})
	return err
}

// updateUbStatusCache updates the user status cache.
func updateUbStatusCache(chatId, userId int64, status string) {
	call, err := vc.Calls.GetGroupAssistant(chatId)
	if err != nil {
		logger.Error("[updateUbStatusCache] Failed: %v", err)
		return
	}

	ubId := call.App.Me().ID
	if userId == ubId {
		vc.Calls.UpdateMembership(chatId, userId, status)
	}
}

// getStatusFromParticipant gets the status from a participant.
func getStatusFromParticipant(p telegram.ChannelParticipant) string {
	switch p.(type) {
	case *telegram.ChannelParticipantCreator:
		return telegram.Creator
	case *telegram.ChannelParticipantAdmin:
		return telegram.Admin
	case *telegram.ChannelParticipantSelf, *telegram.ChannelParticipantObj:
		return telegram.Member
	case *telegram.ChannelParticipantLeft:
		return telegram.Left
	case *telegram.ChannelParticipantBanned:
		return telegram.Kicked
	case nil:
		return telegram.Left
	default:
		logger.Warn("Unknown participant type: %T", p)
		return telegram.Restricted
	}
}

// getChatType gets the chat type from a channel.
func getChatType(ch *telegram.Channel) string {
	if ch.Broadcast {
		return "Broadcast Channel"
	}
	if ch.Megagroup {
		return "Supergroup"
	}
	if ch.Gigagroup {
		return "Gigagroup"
	}
	return "Channel"
}
