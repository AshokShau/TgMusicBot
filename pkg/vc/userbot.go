package vc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// joinAssistant ensures the assistant is a member of the specified chat.
// It checks the user's status and attempts to join or unban if necessary.
func (c *TelegramCalls) joinAssistant(chatID, ubID int64) error {
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	status, err := c.checkUserStats(chatID)
	if err != nil {
		return fmt.Errorf(lang.GetString(langCode, "check_user_status_fail"), err)
	}

	gologging.InfoF("[TelegramCalls - joinAssistant] Chat %d status is: %s", chatID, status)
	switch status {
	case tg.Member, tg.Admin, tg.Creator:
		return nil // The assistant is already in the chat.

	case tg.Left:
		gologging.InfoF("[TelegramCalls - joinAssistant] The assistant is not in the chat; attempting to join...")
		return c.joinUb(chatID)

	case tg.Kicked, tg.Restricted:
		isMuted := status == tg.Restricted
		isBanned := status == tg.Kicked
		gologging.InfoF("[TelegramCalls - joinAssistant] The assistant appears to be %s. Attempting to unban and rejoin...", status)
		botStatus, err := cache.GetUserAdmin(c.bot, chatID, c.bot.Me().ID, false)
		if err != nil {
			if strings.Contains(err.Error(), "is not an admin in chat") {
				return fmt.Errorf(lang.GetString(langCode, "unban_fail_no_admin"), ubID)
			}
			gologging.WarnF("An error occurred while checking the bot's admin status: %v", err)
			return fmt.Errorf(lang.GetString(langCode, "check_admin_status_fail"), err)
		}

		if botStatus.Status != tg.Admin {
			return fmt.Errorf(lang.GetString(langCode, "unban_fail_bot_not_admin"), ubID)
		}

		if botStatus.Rights != nil && !botStatus.Rights.BanUsers {
			return fmt.Errorf(lang.GetString(langCode, "unban_fail_no_perm"), ubID)
		}

		_, err = c.bot.EditBanned(chatID, ubID, &tg.BannedOptions{Unban: isBanned, Unmute: isMuted})
		if err != nil {
			gologging.WarnF("Failed to unban the assistant: %v", err)
			return fmt.Errorf(lang.GetString(langCode, "unban_fail"), ubID, err)
		}

		if isBanned {
			return c.joinUb(chatID)
		}
		return nil

	default:
		gologging.InfoF("[TelegramCalls - joinAssistant] The user status is unknown: %s; attempting to join.", status)
		return c.joinUb(chatID)
	}
}

// checkUserStats checks the membership status of a user in a given chat.
// It returns the user's status as a string and an error if one occurs.
func (c *TelegramCalls) checkUserStats(chatId int64) (string, error) {
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return "", err
	}

	userId := call.App.Me().ID
	cacheKey := fmt.Sprintf("%d:%d", chatId, userId)

	if cached, ok := c.statusCache.Get(cacheKey); ok {
		return cached, nil
	}

	member, err := c.bot.GetChatMember(chatId, userId)
	if err != nil {
		if strings.Contains(err.Error(), "USER_NOT_PARTICIPANT") {
			c.UpdateMembership(chatId, userId, tg.Left)
			return tg.Left, nil
		}

		gologging.InfoF("[TelegramCalls - checkUserStats] Failed to get the chat member: %+v", err)
		c.UpdateMembership(chatId, userId, tg.Left)
		return tg.Left, nil
	}

	c.UpdateMembership(chatId, userId, member.Status)
	return member.Status, nil
}

// joinUb handles the process of a userbot joining a chat via an invite link.
// It returns an error if the userbot fails to join.
func (c *TelegramCalls) joinUb(chatID int64) error {
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	call, err := c.GetGroupAssistant(chatID)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("%d", chatID)
	var link string
	if cached, ok := c.inviteCache.Get(cacheKey); ok {
		link = cached
	} else {
		inviteLink, err := c.bot.GetChatInviteLink(chatID)
		if err != nil {
			return fmt.Errorf(lang.GetString(langCode, "get_invite_link_fail"), err)
		}

		linkObj, ok := inviteLink.(*tg.ChatInviteExported)
		if !ok {
			return fmt.Errorf(lang.GetString(langCode, "invalid_invite_link_type"), inviteLink)
		}

		link = linkObj.Link
		c.UpdateInviteLink(chatID, link)
	}

	gologging.InfoF("[TelegramCalls - joinUb] The invite link is: %s", link)

	ub := call.App
	_, err = ub.JoinChannel(link)
	if err != nil {
		if strings.Contains(err.Error(), "INVITE_REQUEST_SENT") {
			peer, err := c.bot.ResolvePeer(chatID)
			if err != nil {
				return err
			}

			user, err := c.bot.ResolvePeer(ub.Me().ID)
			if err != nil {
				return err
			}

			var inputUser *tg.InputUserObj
			if inpUser, ok := user.(*tg.InputPeerUser); !ok {
				return errors.New(lang.GetString(langCode, "invalid_user_peer"))
			} else {
				inputUser = &tg.InputUserObj{
					UserID:     inpUser.UserID,
					AccessHash: inpUser.AccessHash,
				}
			}

			_, err = c.bot.MessagesHideChatJoinRequest(true, peer, inputUser)
			if err != nil {
				gologging.WarnF("Failed to hide the chat join request: %v", err)
				return fmt.Errorf(lang.GetString(langCode, "join_request_already_sent"), ub.Me().ID)
			}

			return nil
		}

		if strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
			c.UpdateMembership(chatID, ub.Me().ID, tg.Member)
			return nil
		}

		if strings.Contains(err.Error(), "INVITE_HASH_EXPIRED") {
			return fmt.Errorf(lang.GetString(langCode, "invite_link_expired"), ub.Me().ID)
		}

		gologging.InfoF("Failed to join the channel: %v", err)
		return err
	}

	c.UpdateMembership(chatID, ub.Me().ID, tg.Member)
	return nil
}
