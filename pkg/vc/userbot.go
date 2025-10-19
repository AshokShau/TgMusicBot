package vc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// joinAssistant ensures the assistant is a member of the specified chat.
// It checks the user's status and attempts to join or unban if necessary.
func (c *TelegramCalls) joinAssistant(chatId, ubId int64) error {
	status, err := c.checkUserStats(chatId)
	if err != nil {
		return fmt.Errorf("[TelegramCalls - joinAssistant] Failed to check the user's status: %v", err)
	}

	gologging.InfoF("[TelegramCalls - joinAssistant] Chat %d status is: %s", chatId, status)
	switch status {
	case tg.Member, tg.Admin, tg.Creator:
		return nil // The assistant is already in the chat.

	case tg.Left:
		gologging.InfoF("[TelegramCalls - joinAssistant] The assistant is not in the chat; attempting to join...")
		return c.joinUb(chatId)

	case tg.Kicked, tg.Restricted:
		isMuted := status == tg.Restricted
		isBanned := status == tg.Kicked
		gologging.InfoF("[TelegramCalls - joinAssistant] The assistant appears to be %s. Attempting to unban and rejoin...", status)
		botStatus, err := cache.GetUserAdmin(c.bot, chatId, c.bot.Me().ID, false)
		if err != nil {
			if strings.Contains(err.Error(), "is not an admin in chat") {
				return fmt.Errorf(
					"cannot unban the assistant (<code>%d</code>) because it is banned from this group, and I am not an admin",
					ubId,
				)
			}
			gologging.WarnF("An error occurred while checking the bot's admin status: %v", err)
			return fmt.Errorf("failed to check the assistant's admin status: %v", err)
		}

		if botStatus.Status != tg.Admin {
			return fmt.Errorf(
				"cannot unban or unmute the assistant (<code>%d</code>) because it is banned or restricted, and the bot lacks admin privileges",
				ubId,
			)
		}

		if botStatus.Rights != nil && !botStatus.Rights.BanUsers {
			return fmt.Errorf(
				"cannot unban or unmute the assistant (<code>%d</code>) because it is banned or restricted, and the bot lacks the necessary admin privileges",
				ubId,
			)
		}

		_, err = c.bot.EditBanned(chatId, ubId, &tg.BannedOptions{Unban: isBanned, Unmute: isMuted})
		if err != nil {
			gologging.WarnF("Failed to unban the assistant: %v", err)
			return fmt.Errorf("failed to unban the assistant (<code>%d</code>): %v", ubId, err)
		}

		if isBanned {
			return c.joinUb(chatId)
		}
		return nil

	default:
		gologging.InfoF("[TelegramCalls - joinAssistant] The user status is unknown: %s; attempting to join.", status)
		return c.joinUb(chatId)
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
func (c *TelegramCalls) joinUb(chatId int64) error {
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("%d", chatId)
	var link string
	if cached, ok := c.inviteCache.Get(cacheKey); ok {
		link = cached
	} else {
		inviteLink, err := c.bot.GetChatInviteLink(chatId)
		if err != nil {
			return fmt.Errorf("failed to get the invite link: %v", err)
		}

		linkObj, ok := inviteLink.(*tg.ChatInviteExported)
		if !ok {
			return fmt.Errorf("unexpected invite link type received: %T", inviteLink)
		}

		link = linkObj.Link
		c.UpdateInviteLink(chatId, link)
	}

	gologging.InfoF("[TelegramCalls - joinUb] The invite link is: %s", link)

	ub := call.App
	_, err = ub.JoinChannel(link)
	if err != nil {
		if strings.Contains(err.Error(), "INVITE_REQUEST_SENT") {
			peer, err := c.bot.ResolvePeer(chatId)
			if err != nil {
				return err
			}

			user, err := c.bot.ResolvePeer(ub.Me().ID)
			if err != nil {
				return err
			}

			var inputUser *tg.InputUserObj
			if inpUser, ok := user.(*tg.InputPeerUser); !ok {
				return errors.New("user peer is not a valid user")
			} else {
				inputUser = &tg.InputUserObj{
					UserID:     inpUser.UserID,
					AccessHash: inpUser.AccessHash,
				}
			}

			_, err = c.bot.MessagesHideChatJoinRequest(true, peer, inputUser)
			if err != nil {
				gologging.WarnF("Failed to hide the chat join request: %v", err)
				return fmt.Errorf("my assistant (<code>%d</code>) has already requested to join this group", ub.Me().ID)
			}

			return nil
		}

		if strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
			c.UpdateMembership(chatId, ub.Me().ID, tg.Member)
			return nil
		}

		if strings.Contains(err.Error(), "INVITE_HASH_EXPIRED") {
			return fmt.Errorf("the invite link has expired, or my assistant (<code>%d</code>) is banned from this group", ub.Me().ID)
		}

		gologging.InfoF("Failed to join the channel: %v", err)
		return err
	}

	c.UpdateMembership(chatId, ub.Me().ID, tg.Member)
	return nil
}
