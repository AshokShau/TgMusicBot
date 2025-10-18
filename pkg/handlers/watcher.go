package handlers

import (
	"fmt"
	"log"
	"tgmusic/pkg/core"
	"tgmusic/pkg/core/cache"
	"tgmusic/pkg/core/db"
	"tgmusic/pkg/vc"
	"time"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

// handleVoiceChat handles voice chat updates.
// It takes a telegram.Update object and a telegram client as input.
// It returns an error if any.
func handleVoiceChat(upd telegram.Update, c *telegram.Client) error {
	switch update := upd.(type) {
	case *telegram.UpdateNewChannelMessage:
		if msg, ok := update.Message.(*telegram.MessageService); ok {
			chatId, _ := getPeerId(c, msg.PeerID)
			if action, ok := msg.Action.(*telegram.MessageActionGroupCall); ok {
				if action.Duration == 0 {
					cache.ChatCache.ClearChat(chatId, true)
					_, _ = c.SendMessage(chatId, "🎙️ Video chat started!\nUse /play <song name> to play music.")
				} else {
					log.Printf("Voice chat ended. Duration: %d seconds", action.Duration)
					cache.ChatCache.ClearChat(chatId, true)
					_, _ = c.SendMessage(chatId, "🎧 Video chat ended!\nAll queues cleared.")
				}
			} else {
				log.Printf("Unhandled action type: %T", msg.Action)
			}
		}
	}
	return nil
}

// handleParticipant handles participant updates.
// It takes a telegram.ParticipantUpdate object as input.
// It returns an error if any.
func handleParticipant(pu *telegram.ParticipantUpdate) error {
	if pu == nil || pu.Channel == nil {
		gologging.ErrorF("[handleParticipant] Received nil participant update or nil channel")
		return nil
	}

	client := pu.Client
	chatID, _ := getPeerId(client, pu.Channel.ID)
	userID := pu.UserID()
	chat := pu.Channel

	if chatID > 0 {
		text := fmt.Sprintf(
			"This chat (%d) is not a supergroup yet.\n"+
				"<b>⚠️ Please convert this chat to a supergroup and add me as admin.</b>\n\n"+
				"If you don't know how to convert, use this guide:\n"+
				"🔗 https://te.legra.ph/How-to-Convert-a-Group-to-a-Supergroup-01-02\n\n"+
				"If you have any questions, join our support group:",
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

	gologging.DebugF("[handleParticipant] Update: Old=%T New=%T ChatID=%d UserID=%d", pu.Old, pu.New, chatID, userID)

	oldStatus := getStatusFromParticipant(pu.Old)
	newStatus := getStatusFromParticipant(pu.New)

	gologging.DebugF("[handleParticipant] old=%s new=%s chat=%d user=%d", oldStatus, newStatus, chatID, userID)
	call, err := vc.Calls.GetGroupAssistant(chatID)
	if err != nil {
		gologging.ErrorF("[handleParticipant] Failed to get group assistant: %v", err)
		return nil
	}

	ubID := call.App.Me().ID
	if userID != ubID && userID != client.Me().ID {
		gologging.DebugF("[handleParticipant] Ignoring non-self update for user %d", userID)
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
		gologging.InfoF("bot joined chat %d. Initializing...", chatID)
	}

	if userID == ubID {
		gologging.InfoF("UB joined chat %d. Initializing...", chatID)
	}

	gologging.DebugF("User %d joined chat %d", userID, chatID)
	updateUbStatusCache(chatID, userID, telegram.Member)
	return nil
}

// handleLeaveOrKick handles a user leaving or being kicked from a chat.
// It takes a telegram client, a chat ID, a user ID, and a userbot ID as input.
// It returns an error if any.
func handleLeaveOrKick(client *telegram.Client, chatID, userID, ubId int64) error {
	gologging.DebugF("User %d left or was kicked from %d", userID, chatID)
	if userID == ubId {
		gologging.InfoF("UB left chat %d. Stopping call...", chatID)
		cache.ChatCache.ClearChat(chatID, true)
	}

	if userID == client.Me().ID {
		gologging.InfoF("bot left chat %d. Stopping call...", chatID)
		_ = vc.Calls.Stop(chatID)
	}

	updateUbStatusCache(chatID, userID, telegram.Left)
	return nil
}

// handleBan handles a user being banned from a chat.
// It takes a telegram client, a chat ID, a user ID, and a userbot ID as input.
// It returns an error if any.
func handleBan(client *telegram.Client, chatID, userID, ubId int64) error {
	gologging.DebugF("User %d was banned in chat %d", userID, chatID)
	if userID == ubId {
		gologging.InfoF("The bot (assistant) was banned in chat %d. Stopping any active calls and clearing cache...", chatID)
		cache.ChatCache.ClearChat(chatID, true)

		_, err := client.SendMessage(chatID, fmt.Sprintf(
			"🚫 My assistant has been banned from this chat.\n\n"+
				"All ongoing music playback and related data have been stopped and cleared.\n\n"+
				"If this was a mistake, please unban <code>%d</code> to continue using the music features. 🎶",
			ubId,
		))
		if err != nil {
			gologging.ErrorF("Failed to send ban message in chat %d: %v", chatID, err)
			return err
		}
	}

	if userID == client.Me().ID {
		gologging.InfoF("bot banned in chat %d. Stopping call...", chatID)
		_ = vc.Calls.Stop(chatID)
	}

	updateUbStatusCache(chatID, userID, telegram.Kicked)
	return nil
}

// handleUnban handles a user being unbanned from a chat.
// It takes a chat ID and a user ID as input.
// It returns an error if any.
func handleUnban(chatID, userID int64) error {
	gologging.DebugF("User %d was unbanned in chat %d", userID, chatID)
	updateUbStatusCache(chatID, userID, telegram.Left)
	return nil
}

// updateUbStatusCache updates the userbot status cache.
// It takes a chat ID, a user ID, and a status as input.
func updateUbStatusCache(chatId, userId int64, status string) {
	call, err := vc.Calls.GetGroupAssistant(chatId)
	if err != nil {
		gologging.ErrorF("[updateUbStatusCache] Failed to get group assistant: %v", err)
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
			gologging.InfoF("bot promoted in %d, reloading admin cache", chatID)
			_, _ = cache.GetAdmins(client, chatID, true)
		} else {
			gologging.InfoF("bot demoted in %d, clearing admin cache", chatID)
			cache.ClearAdminCache(chatID)
		}
	} else {
		gologging.DebugF("User %d was %s in %d", userID, action, chatID)
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
		gologging.WarnF("Unknown participant type: %T", p)
		return telegram.Restricted
	}
}
