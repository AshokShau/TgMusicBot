/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/config"
	"fmt"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

const (
	StatusCreator    = telegram.Creator
	StatusAdmin      = telegram.Admin
	StatusMember     = telegram.Member
	StatusLeft       = telegram.Left
	StatusKicked     = telegram.Kicked
	StatusRestricted = telegram.Restricted

	ActionPromoted = "promoted"
	ActionDemoted  = "demoted"
)

// getStatusFromParticipant extracts status from participant object
func getStatusFromParticipant(p telegram.ChannelParticipant) string {
	switch v := p.(type) {

	case *telegram.ChannelParticipantCreator:
		return StatusCreator

	case *telegram.ChannelParticipantAdmin:
		return StatusAdmin

	case *telegram.ChannelParticipantSelf, *telegram.ChannelParticipantObj:
		return StatusMember

	case *telegram.ChannelParticipantLeft:
		return StatusLeft

	case *telegram.ChannelParticipantBanned:
		if v.Left {
			return StatusLeft
		}

		if v.BannedRights != nil {
			return StatusRestricted
		}

		return StatusKicked

	case nil:
		logger.Debug("Participant is nil - default Left")
		return StatusLeft

	default:
		logger.Warnf("Unknown participant: %T", p)
		return StatusRestricted
	}
}

// handleParticipant handles participant updates in Telegram groups/channels.
func handleParticipant(pu *telegram.ParticipantUpdate) error {
	if pu == nil || pu.Channel == nil {
		return nil
	}

	client := pu.Client
	chatID := pu.ChannelID()
	userID := pu.UserID()
	chat := pu.Channel

	go storeChatReference(chatID)

	// Update invite link if channel has username
	if chat.Username != "" {
		inviteLink := fmt.Sprintf("https://t.me/%s", chat.Username)
		logger.Debugf("Updating invite link for chat %d: %s", chatID, inviteLink)
		vc.Calls.UpdateInviteLink(chatID, inviteLink)
	}

	oldStatus := getStatusFromParticipant(pu.Old)
	newStatus := getStatusFromParticipant(pu.New)

	logger.Debugf("Status change: UserID=%d, Old=%s, New=%s, ChatID=%d", userID, oldStatus, newStatus, chatID)
	call, err := vc.Calls.GetGroupAssistant(chatID)
	if err != nil {
		logger.Errorf("Failed to get group assistant for chat %d: %v", chatID, err)
		return telegram.ErrEndGroup
	}

	ubID := call.App.Me().ID
	logger.Debugf("Assistant ID for chat %d: %d", chatID, ubID)

	// Only process updates for bot or assistant
	if !isRelevantUser(userID, client.Me().ID, ubID) {
		logger.Debugf("Ignoring update for irrelevant user %d (BotID=%d, AssistantID=%d)", userID, client.Me().ID, ubID)
		return telegram.ErrEndGroup
	}

	logger.Debugf("Processing status change for relevant user %d in chat %d", userID, chatID)
	return handleParticipantStatusChange(client, chatID, userID, ubID, oldStatus, newStatus, chat)
}

// storeChatReference stores chat reference in database
func storeChatReference(chatID int64) {
	ctx, cancel := db.Ctx()
	defer cancel()

	logger.Debugf("Storing chat reference for chat %d", chatID)
	if err := db.Instance.AddChat(ctx, chatID); err != nil {
		logger.Error("Failed to add chat %d to database: %v", chatID, err)
	}
}

// isRelevantUser checks if the user ID belongs to bot or assistant
func isRelevantUser(userID, botID, assistantID int64) bool {
	return userID == botID || userID == assistantID
}

// handleParticipantStatusChange routes status changes to appropriate handlers
func handleParticipantStatusChange(
	client *telegram.Client,
	chatID int64,
	userID, ubID int64,
	oldStatus, newStatus string,
	channel *telegram.Channel,
) error {

	logger.Debugf("Routing status change: Chat=%d, User=%d, Old=%s, New=%s", chatID, userID, oldStatus, newStatus)

	switch {
	case oldStatus == StatusLeft && (newStatus == StatusMember || newStatus == StatusAdmin):
		return handleJoin(client, chatID, userID, ubID, channel)

	case (oldStatus == StatusMember || oldStatus == StatusAdmin) && newStatus == StatusLeft:
		return handleLeave(client, chatID, userID, ubID)

	case newStatus == StatusKicked:
		return handleBan(client, chatID, userID, ubID)

	case oldStatus == StatusKicked && newStatus == StatusLeft:
		return handleUnban(chatID, userID)

	default:
		return handlePromotionDemotion(client, chatID, userID, oldStatus, newStatus, channel)
	}
}

// handleJoin handles user/bot/assistant joining a chat
func handleJoin(client *telegram.Client, chatID, userID, ubID int64, channel *telegram.Channel) error {
	logger.Infof("User %d joined chat %d", userID, chatID)

	if userID == client.Me().ID {
		logger.Infof("Bot joined chat %d. Initializing bot services...", chatID)
		sendJoinLog(client, chatID, channel)
	}

	if userID == ubID {
		logger.Infof("Assistant joined chat %d. Initializing assistant...", chatID)
	}

	updateStatusCache(chatID, userID, StatusMember)
	return nil
}

// sendJoinLog sends log message to admin when bot joins a new chat
func sendJoinLog(client *telegram.Client, chatID int64, channel *telegram.Channel) {
	chatType := getChatType(channel)

	text := fmt.Sprintf(
		"<b>🤖 Bot Joined a New Chat</b>\n"+
			"📌 <b>Chat ID:</b> <code>%d</code>\n"+
			"🏷️ <b>Title:</b> %s\n"+
			"👥 <b>Type:</b> %s\n"+
			"👤 <b>Username:</b> @%s\n",
		chatID,
		channel.Title,
		chatType,
		channel.Username,
	)

	_, err := client.SendMessage(config.Conf.LoggerId, text, &telegram.SendOptions{
		LinkPreview: false,
	})

	if err != nil {
		logger.Warnf("Failed to send join log to admin: %v", err)
	}
}

// handleLeave handles user/bot/assistant leaving a chat
func handleLeave(client *telegram.Client, chatID, userID, ubID int64) error {
	logger.Infof("User %d left chat %d", userID, chatID)

	if userID == ubID {
		logger.Infof("Assistant left chat %d. Clearing cache...", chatID)
		cache.ChatCache.ClearChat(chatID)
	}

	if userID == client.Me().ID {
		logger.Infof("Bot left chat %d. Stopping voice call...", chatID)
		if err := vc.Calls.Stop(chatID); err != nil {
			logger.Errorf("Failed to stop voice call for chat %d: %v", chatID, err)
		}
	}

	updateStatusCache(chatID, userID, StatusLeft)
	return nil
}

// handleBan handles user/bot/assistant being banned from chat
func handleBan(client *telegram.Client, chatID, userID, ubID int64) error {
	logger.Debugf("User %d was banned from chat %d", userID, chatID)

	if userID == ubID {
		logger.Warnf("Assistant banned from chat %d. Performing cleanup...", chatID)
		cache.ChatCache.ClearChat(chatID)

		message := fmt.Sprintf("🚫 My assistant has been banned from this chat.\n\nAll ongoing music playback and related data have been stopped and cleared.\n\nIf this was a mistake, please unban <code>%d</code> to continue using the music features. 🎶", ubID)
		_, err := client.SendMessage(chatID, message)
		if err != nil {
			logger.Errorf("Failed to send assistant ban notification: %v", err)
			return err
		}

		logger.Infof("Assistant ban notification sent to chat %d", chatID)
	}

	if userID == client.Me().ID {
		logger.Warnf("Bot was banned from chat %d. Stopping services...", chatID)
		if err := vc.Calls.Stop(chatID); err != nil {
			logger.Errorf("Failed to stop voice call after ban: %d (%v)", chatID, err)
		}
	}

	updateStatusCache(chatID, userID, StatusKicked)
	return nil
}

// handleUnban handles user/bot/assistant being unbanned
func handleUnban(chatID, userID int64) error {
	logger.Infof("User %d was unbanned from chat %d", userID, chatID)
	updateStatusCache(chatID, userID, StatusLeft)
	return nil
}

// handlePromotionDemotion handles admin promotion/demotion events
func handlePromotionDemotion(
	client *telegram.Client,
	chatID, userID int64,
	oldStatus, newStatus string,
	channel *telegram.Channel,
) error {

	isPromoted := oldStatus != StatusAdmin && newStatus == StatusAdmin
	isDemoted := oldStatus == StatusAdmin && newStatus != StatusAdmin

	if !isPromoted && !isDemoted {
		logger.Debugf("Not a promotion/demotion event for user %d in chat %d", userID, chatID)
		return nil
	}

	action := ActionPromoted
	if isDemoted {
		action = ActionDemoted
	}

	logger.Infof("User %d was %s in chat %d", userID, action, chatID)
	if userID == client.Me().ID {
		handleBotAdminChange(client, chatID, isPromoted)
		if err := sendAdminStatusLog(client, chatID, userID, action, channel); err != nil {
			logger.Errorf("Failed to send admin status log: %v", err)
		}
	}

	vc.Calls.UpdateMembership(chatID, userID, newStatus)
	return nil
}

// handleBotAdminChange handles bot's admin status changes
func handleBotAdminChange(client *telegram.Client, chatID int64, isPromoted bool) {
	if isPromoted {
		logger.Infof("Bot promoted in chat %d. Refreshing admin cache...", chatID)
		admins, err := cache.GetAdmins(client, chatID, true)
		if err != nil {
			logger.Errorf("Failed to refresh admin cache: %v", err)
		} else {
			logger.Debugf("Admin cache refreshed. Found %d admins", len(admins))
		}
	} else {
		logger.Warnf("Bot demoted in chat %d. Clearing admin cache...", chatID)
		cache.ClearAdminCache(chatID)
	}
}

// sendAdminStatusLog sends admin status change notification to logger channel
func sendAdminStatusLog(client *telegram.Client, chatID, userID int64, action string, ch *telegram.Channel) error {
	logger.Debugf("Sending admin status log for user %d in chat %d", userID, chatID)

	text := fmt.Sprintf(
		"<b>⚠️ Admin Status Changed</b>\n"+
			"📌 <b>Chat:</b> %s (<code>%d</code>)\n"+
			"👤 <b>User:</b> <code>%d</code>\n"+
			"🔧 <b>Action:</b> %s\n",
		ch.Title,
		chatID,
		userID,
		action,
	)

	_, err := client.SendMessage(config.Conf.LoggerId, text, &telegram.SendOptions{
		LinkPreview: false,
	})

	return err
}

// updateStatusCache updates the user status in cache
func updateStatusCache(chatID, userID int64, status string) {
	logger.Debugf("Updating status cache: Chat=%d, User=%d, Status=%s", chatID, userID, status)
	call, err := vc.Calls.GetGroupAssistant(chatID)
	if err != nil {
		logger.Errorf("Failed to get group assistant for cache update: %v", err)
		return
	}

	ubID := call.App.Me().ID
	if userID == ubID {
		logger.Debugf("Updating assistant membership: Chat=%d, Assistant=%d, Status=%s", chatID, ubID, status)
		vc.Calls.UpdateMembership(chatID, userID, status)
	}
}

// getChatType returns a human-readable chat type string
func getChatType(ch *telegram.Channel) string {
	if ch.Broadcast {
		return "Broadcast Channel"
	} else if ch.Megagroup {
		return "Supergroup"
	} else if ch.Gigagroup {
		return "Gigagroup"
	}
	return "Channel"
}
