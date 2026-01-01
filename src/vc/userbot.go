/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
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
	"time"

	"ashokshau/tgmusic/src/core/cache"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// joinAssistant ensures the assistant is a member of the specified chat.
func (c *TelegramCalls) joinAssistant(chatID, ubID int64) error {
	status, err := c.checkUserStats(chatID)
	if err != nil {
		return fmt.Errorf("[TelegramCalls - joinAssistant] Failed to check the user's status: %v", err)
	}

	logger.Info("Chat %d status is: %s", chatID, status)
	switch status {
	case tg.Member, tg.Admin, tg.Creator:
		return nil // The assistant is already in the chat.

	case tg.Left:
		logger.Info("The assistant is not in the chat; attempting to join...")
		return c.joinUb(chatID)

	case tg.Kicked, tg.Restricted:
		isMuted := status == tg.Restricted
		isBanned := status == tg.Kicked
		logger.Infof("The assistant appears to be %s. Attempting to unban and rejoin...", status)
		botStatus, err := cache.GetUserAdmin(c.bot, chatID, c.bot.Me().ID, false)
		if err != nil {
			if strings.Contains(err.Error(), "is not an admin in chat") {
				return fmt.Errorf("cannot unban the assistant (<code>%d</code>) because it is banned from this group, and I am not an admin", ubID)
			}

			logger.Warnf("An error occurred while checking the bot's admin status: %v", err)
			return fmt.Errorf("failed to check the assistant's admin status: %v", err)
		}

		if botStatus.Status != tg.Admin {
			return fmt.Errorf("cannot unban or unmute the assistant (<code>%d</code>) because it is banned or restricted, and the bot lacks admin privileges", ubID)
		}

		if botStatus.Rights != nil && !botStatus.Rights.BanUsers {
			return fmt.Errorf("cannot unban or unmute the assistant (<code>%d</code>) because it is banned or restricted, and the bot lacks the necessary admin privileges", ubID)
		}

		_, err = c.bot.EditBanned(chatID, ubID, &tg.BannedOptions{Unban: isBanned, Unmute: isMuted})
		if err != nil {
			logger.Warnf("Failed to unban the assistant: %v", err)
			return fmt.Errorf("failed to unban the assistant (<code>%d</code>): %v", ubID, err)
		}

		if isBanned {
			return c.joinUb(chatID)
		}

		return nil

	default:
		logger.Warnf("The user status is unknown: %s; attempting to join.", status)
		return c.joinUb(chatID)
	}
}

// checkUserStats checks the membership status of a user in a given chat.
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

		logger.Info("Failed to get the chat member: %+v", err)
		c.UpdateMembership(chatId, userId, tg.Left)
		return tg.Left, nil
	}

	c.UpdateMembership(chatId, userId, member.Status)
	return member.Status, nil
}

// joinUb handles the process of a user-bot joining a chat via an invite link.
func (c *TelegramCalls) joinUb(chatID int64) error {
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
				Peer:          peer,
				Title:         "TgMusicBot Assistant",
				RequestNeeded: false,
			})

			if err != nil {
				logger.Warnf("Failed to export invite link: %v", err)
				return fmt.Errorf("failed to get the invite link: %v", err)
			}

			exported, ok := raw.(*tg.ChatInviteExported)
			if !ok || exported.Link == "" {
				return fmt.Errorf("unexpected invite link type received: %T", raw)
			}

			link = exported.Link
		}

		if link == "" {
			logger.Warn("Failed to get or create invite link")
			return errors.New("failed to get/create invite link")
		}

		c.UpdateInviteLink(chatID, link)
	}

	logger.Infof("Using invite link: %s", link)
	_, err = ub.JoinChannel(link)
	if err != nil {
		errStr := err.Error()
		userID := ub.Me().ID

		switch {
		case strings.Contains(errStr, "INVITE_REQUEST_SENT"):
			time.Sleep(1 * time.Second)
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
				return errors.New("user peer is not a valid user")
			}

			inputUser := &tg.InputUserObj{
				UserID:     inpUser.UserID,
				AccessHash: inpUser.AccessHash,
			}

			if _, err := c.bot.MessagesHideChatJoinRequest(true, peer, inputUser); err != nil {
				if strings.Contains(err.Error(), "HIDE_REQUESTER_MISSIN") {
					c.UpdateMembership(chatID, userID, tg.Member)
					return nil
				}

				logger.Warnf("Failed to hide chat join request: %v", err)
				return fmt.Errorf(
					"my assistant (<code>%d</code>) has already requested to join this group",
					userID,
				)
			}

			return nil

		case strings.Contains(errStr, "USER_ALREADY_PARTICIPANT"):
			c.UpdateMembership(chatID, userID, tg.Member)
			return nil

		case strings.Contains(errStr, "INVITE_HASH_EXPIRED"):
			return fmt.Errorf(
				"the invite link has expired, or my assistant (<code>%d</code>) is banned from this group",
				userID,
			)

		case strings.Contains(errStr, "CHANNEL_PRIVATE"):
			c.UpdateMembership(chatID, userID, tg.Left)
			c.UpdateInviteLink(chatID, "")
			return fmt.Errorf("my assistant (<code>%d</code>) is banned from this group", userID)
		}

		logger.Infof("Failed to join channel: %v", err)
		return err
	}

	c.UpdateMembership(chatID, ub.Me().ID, tg.Member)
	return nil
}
