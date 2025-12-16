/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package vc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"

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

	logger.Info("[TelegramCalls - joinAssistant] Chat %d status is: %s", chatID, status)
	switch status {
	case tg.Member, tg.Admin, tg.Creator:
		return nil // The assistant is already in the chat.

	case tg.Left:
		logger.Info("[TelegramCalls - joinAssistant] The assistant is not in the chat; attempting to join...")
		return c.joinUb(chatID)

	case tg.Kicked, tg.Restricted:
		isMuted := status == tg.Restricted
		isBanned := status == tg.Kicked
		logger.Info("[TelegramCalls - joinAssistant] The assistant appears to be %s. Attempting to unban and rejoin...", status)
		botStatus, err := cache.GetUserAdmin(c.bot, chatID, c.bot.Me().ID, false)
		if err != nil {
			if strings.Contains(err.Error(), "is not an admin in chat") {
				return fmt.Errorf(lang.GetString(langCode, "unban_fail_no_admin"), ubID)
			}
			logger.Warn("An error occurred while checking the bot's admin status: %v", err)
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
			logger.Warn("Failed to unban the assistant: %v", err)
			return fmt.Errorf(lang.GetString(langCode, "unban_fail"), ubID, err)
		}

		if isBanned {
			return c.joinUb(chatID)
		}

		return nil

	default:
		logger.Info("[TelegramCalls - joinAssistant] The user status is unknown: %s; attempting to join.", status)
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

		logger.Info("[TelegramCalls - checkUserStats] Failed to get the chat member: %+v", err)
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
	ub := call.App
	cacheKey := strconv.FormatInt(chatID, 10)
	link := ""

	if cached, ok := c.inviteCache.Get(cacheKey); ok && cached != "" {
		link = cached
	} else {
		raw, err := c.bot.GetChatInviteLink(chatID)
		if err == nil {
			if exported, ok := raw.(*tg.ChatInviteExported); ok && exported.Link != "" {
				link = exported.Link
			}
		}

		if link == "" {
			peer, err := c.bot.ResolvePeer(chatID)
			if err != nil {
				return errors.New("failed to resolve peer")
			}

			raw, err = c.bot.MessagesExportChatInvite(&tg.MessagesExportChatInviteParams{
				Peer:  peer,
				Title: "for my ub",
			})
			if err != nil {
				logger.Warnf("Failed to export invite link: %v", err)
				return fmt.Errorf(lang.GetString(langCode, "get_invite_link_fail"), err)
			}

			exported, ok := raw.(*tg.ChatInviteExported)
			if !ok || exported.Link == "" {
				return fmt.Errorf(lang.GetString(langCode, "invalid_invite_link_type"), raw)
			}

			link = exported.Link
		}

		if link == "" {
			logger.Warn("[joinUb] Failed to get or create invite link")
			return errors.New("failed to get/create invite link")
		}

		c.UpdateInviteLink(chatID, link)
	}

	logger.Infof("[joinUb] Using invite link: %s", link)
	_, err = ub.JoinChannel(link)
	if err != nil {
		errStr := err.Error()
		userID := ub.Me().ID

		switch {
		case strings.Contains(errStr, "INVITE_REQUEST_SENT"):
			peer, err := c.bot.ResolvePeer(chatID)
			if err != nil {
				return err
			}

			userPeer, err := c.bot.ResolvePeer(userID)
			if err != nil {
				return err
			}

			inpUser, ok := userPeer.(*tg.InputPeerUser)
			if !ok {
				return errors.New(lang.GetString(langCode, "invalid_user_peer"))
			}

			inputUser := &tg.InputUserObj{
				UserID:     inpUser.UserID,
				AccessHash: inpUser.AccessHash,
			}

			if _, err := c.bot.MessagesHideChatJoinRequest(true, peer, inputUser); err != nil {
				logger.Warnf("Failed to hide chat join request: %v", err)
				return fmt.Errorf(
					lang.GetString(langCode, "join_request_already_sent"),
					userID,
				)
			}

			return nil

		case strings.Contains(errStr, "USER_ALREADY_PARTICIPANT"):
			c.UpdateMembership(chatID, userID, tg.Member)
			return nil

		case strings.Contains(errStr, "INVITE_HASH_EXPIRED"):
			return fmt.Errorf(
				lang.GetString(langCode, "invite_link_expired"),
				userID,
			)

		case strings.Contains(errStr, "CHANNEL_PRIVATE"):
			c.UpdateMembership(chatID, userID, tg.Left)
			c.UpdateInviteLink(chatID, "")
			return nil
		}

		logger.Infof("Failed to join channel: %v", err)
		return err
	}

	c.UpdateMembership(chatID, ub.Me().ID, tg.Member)
	return nil
}
