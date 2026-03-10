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
	"log/slog"
	"strconv"
	"strings"
	"time"

	"ashokshau/tgmusic/src/core/cache"

	td "github.com/AshokShau/gotdbot"
)

// joinAssistant ensures the assistant is a member of the specified chat.
func (c *TelegramCalls) joinAssistant(chatID, ubID int64) error {
	status, err := c.checkUserStats(chatID)
	if err != nil {
		return fmt.Errorf("[TelegramCalls - joinAssistant] Failed to check the user's status: %v", err)
	}

	logger.Info("Chat  status is", "chat_id", chatID, "arg2", status)
	switch status {
	case td.ChatMemberStatusCreator{}, td.ChatMemberStatusAdministrator{}, td.ChatMemberStatusMember{}:
		return nil // The assistant is already in the chat.

	case td.ChatMemberStatusLeft{}:
		logger.Info("The assistant is not in the chat; attempting to join...")
		return c.joinUb(chatID)

	case td.ChatMemberStatusBanned{}, td.ChatMemberStatusRestricted{}:
		//isMuted := status == td.ChatMemberStatusRestricted{}
		isBanned := status == td.ChatMemberStatusBanned{}
		logger.Info("The assistant appears to be . Attempting to unban and rejoin...", "arg1", status)
		botStatus, err := cache.GetUserAdmin(c.bot, chatID, c.bot.Me().Id, false)
		if err != nil {
			if strings.Contains(err.Error(), "is not an admin in chat") {
				return fmt.Errorf("cannot unban the assistant (<code>%d</code>) because it is banned from this group, and I am not an admin", ubID)
			}

			logger.Warn("An error occurred while checking the bot's admin status", "error", err)
			return fmt.Errorf("failed to check the assistant's admin status: %v", err)
		}

		admin, ok := botStatus.Status.(*td.ChatMemberStatusAdministrator)
		if !ok {
			return fmt.Errorf(
				"cannot unban or unmute the assistant (<code>%d</code>) because it is banned or restricted, and the bot lacks admin privileges",
				ubID,
			)
		}

		if admin.Rights == nil || !admin.Rights.CanRestrictMembers {
			return fmt.Errorf(
				"cannot unban or unmute the assistant (<code>%d</code>) because it is banned or restricted, and the bot lacks the necessary admin privileges",
				ubID,
			)
		}

		err = c.bot.SetChatMemberStatus(chatID, td.MessageSenderUser{UserId: ubID}, &td.ChatMemberStatusMember{})
		if err != nil {
			logger.Warn("Failed to unban the assistant", "error", err)
			return fmt.Errorf("failed to unban the assistant (<code>%d</code>): %v", ubID, err)
		}

		if isBanned {
			return c.joinUb(chatID)
		}

		return nil

	default:
		logger.Warn("The user status is unknown: ; attempting to join.", "arg1", status)
		return c.joinUb(chatID)
	}
}

// checkUserStats checks the membership status of a user in a given chat.
func (c *TelegramCalls) checkUserStats(chatId int64) (td.ChatMemberStatus, error) {
	call, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return nil, err
	}

	userId := call.App.Me().ID
	cacheKey := fmt.Sprintf("%d:%d", chatId, userId)

	if cached, ok := c.statusCache.Get(cacheKey); ok {
		return cached, nil
	}

	member, err := c.bot.GetChatMember(chatId, td.MessageSenderUser{UserId: userId})
	if err != nil {
		if strings.Contains(err.Error(), "USER_NOT_PARTICIPANT") {
			c.UpdateMembership(chatId, userId, td.ChatMemberStatusLeft{})
			return td.ChatMemberStatusLeft{}, nil
		}

		logger.Info("Failed to get the chat member", "error", err)
		c.UpdateMembership(chatId, userId, td.ChatMemberStatusLeft{})
		return td.ChatMemberStatusLeft{}, nil
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
		chatLink, err := c.bot.CreateChatInviteLink(chatID, 0, 5, "FallenBeatz", &td.CreateChatInviteLinkOpts{CreatesJoinRequest: false})
		if err != nil {
			logger.Warn("Failed to create invite link", "error", err)
			return fmt.Errorf("failed to create invite link: %v", err)
		}

		link = chatLink.InviteLink
		if link == "" {
			logger.Warn("Failed to get or create invite link")
			return errors.New("failed to get/create invite link")
		}

		c.UpdateInviteLink(chatID, link)
	}

	logger.Info("Using invite link", "arg1", link)
	_, err = ub.JoinChannel(link)
	if err != nil {
		errStr := err.Error()
		userID := ub.Me().ID

		switch {
		case strings.Contains(errStr, "INVITE_REQUEST_SENT"):
			time.Sleep(1 * time.Second)
			err = c.bot.ProcessChatJoinRequest(chatID, userID, &td.ProcessChatJoinRequestOpts{Approve: true})
			if err != nil {
				slog.Info("Failed to approve chat join request", "error", err)
				return fmt.Errorf(
					"my assistant (<code>%d</code>) has already requested to join this group",
					userID,
				)
			}

			return nil

		case strings.Contains(errStr, "USER_ALREADY_PARTICIPANT"):
			c.UpdateMembership(chatID, userID, td.ChatMemberStatusMember{})
			return nil

		case strings.Contains(errStr, "INVITE_HASH_EXPIRED"):
			return fmt.Errorf(
				"the invite link has expired, or my assistant (<code>%d</code>) is banned from this group",
				userID,
			)

		case strings.Contains(errStr, "CHANNEL_PRIVATE"):
			c.UpdateMembership(chatID, userID, td.ChatMemberStatusLeft{})
			c.UpdateInviteLink(chatID, "")
			return fmt.Errorf("my assistant (<code>%d</code>) is banned from this group", userID)
		}

		logger.Info("Failed to join channel", "error", err)
		return err
	}

	c.UpdateMembership(chatID, ub.Me().ID, td.ChatMemberStatusMember{})
	return nil
}
