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
	"time"

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"
	"ashokshau/tgmusic/src/vc"

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

	if action, ok := m.Action.(*telegram.MessageActionGroupCall); ok {
		message := ""
		if action.Duration == 0 {
			if cache.ChatCache != nil {
				cache.ChatCache.ClearChat(chatID, true)
			}
			message = lang.GetString(langCode, "watcher_vc_started")
		} else {
			logger.Info("Voice chat ended. Duration: %d seconds", action.Duration)
			if cache.ChatCache != nil {
				cache.ChatCache.ClearChat(chatID, true)
			}
			message = lang.GetString(langCode, "watcher_vc_ended")
		}

		if message != "" {
			_, _ = m.Client.SendMessage(chatID, message)
		}
	} else {
		logger.Info("Unhandled action type: %T", m.Action)
	}

	return nil
}

// handleParticipant handles participant updates.
// It takes a telegram.ParticipantUpdate object as input.
// It returns an error if any.
func handleParticipant(pu *telegram.ParticipantUpdate) error {
	if pu == nil || pu.Channel == nil {
		logger.Error("[handleParticipant] Received nil participant update or nil channel")
		return nil
	}

	client := pu.Client
	chatID := pu.ChannelID()

	userID := pu.UserID()
	chat := pu.Channel
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if chatID > 0 {
		text := fmt.Sprintf(lang.GetString(langCode, "watcher_not_supergroup"),
			chatID,
		)

		_, _ = client.SendMessage(chatID, text, &telegram.SendOptions{
			ReplyMarkup: core.AddMeMarkup(client.Me().Username),
			LinkPreview: false,
		})

		time.Sleep(1 * time.Second)
		_ = client.LeaveChannel(pu.ChannelID())
		return nil
	}

	go func(chatID int64) {
		ctx, cancel := db.Ctx()
		defer cancel()
		_ = db.Instance.AddChat(ctx, chatID)
	}(chatID)

	if chat.Username != "" {
		vc.Calls.UpdateInviteLink(chatID, fmt.Sprintf("https://t.me/%s", chat.Username))
	}

	logger.Debug("[handleParticipant] Update: Old=%T New=%T ChatID=%d UserID=%d", pu.Old, pu.New, chatID, userID)

	oldStatus := getStatusFromParticipant(pu.Old)
	newStatus := getStatusFromParticipant(pu.New)

	logger.Debug("[handleParticipant] old=%s new=%s chat=%d user=%d", oldStatus, newStatus, chatID, userID)
	call, err := vc.Calls.GetGroupAssistant(chatID)
	if err != nil {
		logger.Error("[handleParticipant] Failed to get group assistant: %v", err)
		return nil
	}

	ubID := call.App.Me().ID
	if userID != ubID && userID != client.Me().ID {
		logger.Debug("[handleParticipant] Ignoring non-self update for user %d", userID)
		return nil
	}

	return handleParticipantStatusChange(client, chatID, userID, ubID, oldStatus, newStatus)
}

// handleParticipantStatusChange handles participant status changes.
// It takes a telegram client, a chat ID, a user ID, a userbot ID, the old status, and the new status as input.
// It returns an error if any.
func handleParticipantStatusChange(client *telegram.Client, chatID int64, userID, ubID int64, oldStatus, newStatus string) error {
	switch {
	case oldStatus == telegram.Left && (newStatus == telegram.Member || newStatus == telegram.Admin):
		return handleJoin(client, chatID, userID, ubID)
	case (oldStatus == telegram.Member || oldStatus == telegram.Admin) && newStatus == telegram.Left:
		return handleLeaveOrKick(client, chatID, userID, ubID)
	case newStatus == telegram.Kicked:
		return handleBan(client, chatID, userID, ubID)
	case oldStatus == telegram.Kicked && newStatus == telegram.Left:
		return handleUnban(chatID, userID)
	default:
		return handlePromotionDemotion(client, chatID, userID, oldStatus, newStatus)
	}
}

// handleJoin handles a user joining a chat.
// It takes a telegram client, a chat ID, a user ID, and a userbot ID as input.
// It returns an error if any.
func handleJoin(client *telegram.Client, chatID, userID, ubID int64) error {
	if userID == client.Me().ID {
		logger.Info("bot joined chat %d. Initializing...", chatID)
	}

	if userID == ubID {
		logger.Info("UB joined chat %d. Initializing...", chatID)
	}

	logger.Debug("User %d joined chat %d", userID, chatID)
	updateUbStatusCache(chatID, userID, telegram.Member)
	return nil
}

// handleLeaveOrKick handles a user leaving or being kicked from a chat.
// It takes a telegram client, a chat ID, a user ID, and a userbot ID as input.
// It returns an error if any.
func handleLeaveOrKick(client *telegram.Client, chatID, userID, ubId int64) error {
	logger.Debug("User %d left or was kicked from %d", userID, chatID)
	if userID == ubId {
		logger.Info("UB left chat %d. Stopping call...", chatID)
		cache.ChatCache.ClearChat(chatID, true)
	}

	if userID == client.Me().ID {
		logger.Info("bot left chat %d. Stopping call...", chatID)
		_ = vc.Calls.Stop(chatID)
	}

	updateUbStatusCache(chatID, userID, telegram.Left)
	return nil
}

// handleBan handles a user being banned from a chat.
// It takes a telegram client, a chat ID, a user ID, and a userbot ID as input.
// It returns an error if any.
func handleBan(client *telegram.Client, chatID, userID, ubId int64) error {
	logger.Debug("User %d was banned in chat %d", userID, chatID)
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if userID == ubId {
		logger.Info("The bot (assistant) was banned in chat %d. Stopping any active calls and clearing cache...", chatID)
		cache.ChatCache.ClearChat(chatID, true)

		_, err := client.SendMessage(chatID, fmt.Sprintf(lang.GetString(langCode, "watcher_assistant_banned"),
			ubId,
		))
		if err != nil {
			logger.Error("Failed to send ban message in chat %d: %v", chatID, err)
			return err
		}
	}

	if userID == client.Me().ID {
		logger.Info("bot banned in chat %d. Stopping call...", chatID)
		_ = vc.Calls.Stop(chatID)
	}

	updateUbStatusCache(chatID, userID, telegram.Kicked)
	return nil
}

// handleUnban handles a user being unbanned from a chat.
// It takes a chat ID and a user ID as input.
// It returns an error if any.
func handleUnban(chatID, userID int64) error {
	logger.Debug("User %d was unbanned in chat %d", userID, chatID)
	updateUbStatusCache(chatID, userID, telegram.Left)
	return nil
}

// updateUbStatusCache updates the userbot status cache.
// It takes a chat ID, a user ID, and a status as input.
func updateUbStatusCache(chatId, userId int64, status string) {
	call, err := vc.Calls.GetGroupAssistant(chatId)
	if err != nil {
		logger.Error("[updateUbStatusCache] Failed to get group assistant: %v", err)
		return
	}

	ubId := call.App.Me().ID
	if userId == ubId {
		vc.Calls.UpdateMembership(chatId, userId, status)
	}
}

// handlePromotionDemotion handles a user being promoted or demoted.
// It takes a telegram client, a chat ID, a user ID, the old status, and the new status as input.
// It returns an error if any.
func handlePromotionDemotion(client *telegram.Client, chatID, userID int64, oldStatus, newStatus string) error {
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
			logger.Info("bot promoted in %d, reloading admin cache", chatID)
			_, _ = cache.GetAdmins(client, chatID, true)
		} else {
			logger.Info("bot demoted in %d, clearing admin cache", chatID)
			cache.ClearAdminCache(chatID)
		}
	} else {
		logger.Debug("User %d was %s in %d", userID, action, chatID)
	}

	vc.Calls.UpdateMembership(chatID, userID, newStatus)
	return nil
}

// getStatusFromParticipant gets the status from a participant.
// It takes a telegram.ChannelParticipant object as input.
// It returns the status of the participant as a string.
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
