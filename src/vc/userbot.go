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
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/vc/ubot"

	td "github.com/AshokShau/gotdbot"
)

// joinAssistant ensures the assistant is a member of the specified chat.
func (c *TelegramCalls) joinAssistant(chatID, ubID int64) error {
	status, err := c.checkUserStats(chatID)
	if err != nil {
		return fmt.Errorf("joinAssistant: check user status: %w", err)
	}

	logger.Info("chat member status", "chat_id", chatID, "status", status)

	switch status.(type) {
	case *td.ChatMemberStatusCreator,
		*td.ChatMemberStatusAdministrator,
		*td.ChatMemberStatusMember:
		return nil

	case *td.ChatMemberStatusLeft:
		logger.Info("assistant is not in chat, joining", "chat_id", chatID)
		return c.joinUb(chatID)

	case *td.ChatMemberStatusBanned, *td.ChatMemberStatusRestricted:
		_, isBanned := status.(*td.ChatMemberStatusBanned)
		_, isMuted := status.(*td.ChatMemberStatusRestricted)
		logger.Info("assistant is banned or restricted, attempting recovery",
			"chat_id", chatID, "banned", isBanned, "muted", isMuted)

		return c.recoverBannedAssistant(chatID, ubID, isBanned)

	default:
		logger.Warn("unknown assistant status, attempting to join", "status", status)
		return c.joinUb(chatID)
	}
}

// recoverBannedAssistant attempts to unban or unmute the assistant using bot admin rights.
func (c *TelegramCalls) recoverBannedAssistant(chatID, ubID int64, isBanned bool) error {
	botStatus, err := cache.GetUserAdmin(c.bot, chatID, c.bot.Me().Id, false)
	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			return fmt.Errorf(
				"cannot unban assistant (<code>%d</code>): it is banned and the bot is not an admin",
				ubID,
			)
		}
		return fmt.Errorf("failed to check bot admin status: %w", err)
	}

	admin, ok := botStatus.Status.(*td.ChatMemberStatusAdministrator)
	if !ok || admin.Rights == nil || !admin.Rights.CanRestrictMembers {
		return fmt.Errorf(
			"cannot unban/unmute assistant (<code>%d</code>): bot lacks CanRestrictMembers",
			ubID,
		)
	}

	if isBanned {
		if err := c.bot.SetChatMemberStatus(
			chatID,
			td.MessageSenderUser{UserId: ubID},
			&td.ChatMemberStatusMember{},
		); err != nil {
			logger.Warn("failed to unban assistant", "ub_id", ubID, "error", err)
		}
		return c.joinUb(chatID)
	}

	// isMuted: restricted but not banned — nothing actionable right now.
	// TODO: call SetChatMemberStatus to lift restrictions.
	return nil
}

// JoinAssistant attempts to join an assistant to the chat. If the assigned assistant
// fails, it iterates through all available assistants until one succeeds or all fail.
func (c *TelegramCalls) JoinAssistant(chatID int64) (*ubot.Context, error) {
	ctx, cancel := db.Ctx()
	defer cancel()

	c.mu.RLock()
	totalClients := len(c.availableClients)
	c.mu.RUnlock()

	if totalClients == 0 {
		return nil, errors.New("no clients are available")
	}

	var excludedClients []string

	for i := 0; i < totalClients; i++ {
		call, err := c.GetGroupAssistant(chatID, excludedClients...)
		if err != nil {
			return nil, err
		}

		assistantID := call.App.Me().ID

		if err = c.joinAssistant(chatID, assistantID); err != nil {
			slog.Info("assistant failed to join chat",
				"chat_id", chatID, "assistant_id", assistantID, "error", err)

			// Find this client's name to exclude it on the next iteration.
			if name := c.clientNameFor(call); name != "" {
				excludedClients = append(excludedClients, name)
			}

			cacheKey := fmt.Sprintf("%d:%d", chatID, assistantID)
			c.statusCache.Delete(cacheKey)
			_ = db.Instance.RemoveAssistant(ctx, chatID)

			if i == totalClients-1 {
				return nil, err // all assistants exhausted
			}
			continue
		}

		return call, nil
	}

	// Unreachable, but satisfies the compiler.
	return nil, errors.New("all assistants failed to join")
}

// clientNameFor returns the uBContext key for the given call, or "" if not found.
// Caller must not hold mu.
func (c *TelegramCalls) clientNameFor(call *ubot.Context) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for name, ctx := range c.uBContext {
		if ctx == call {
			return name
		}
	}
	return ""
}

// checkUserStats returns the assistant's membership status in chatID.
// Results are cached; a cache miss triggers a live Telegram API call.
func (c *TelegramCalls) checkUserStats(chatID int64) (td.ChatMemberStatus, error) {
	call, err := c.GetGroupAssistant(chatID)
	if err != nil {
		return nil, err
	}

	userID := call.App.Me().ID
	cacheKey := fmt.Sprintf("%d:%d", chatID, userID)

	if cached, ok := c.statusCache.Get(cacheKey); ok {
		return cached, nil
	}

	member, err := c.bot.GetChatMember(chatID, td.MessageSenderUser{UserId: userID})
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "USER_NOT_PARTICIPANT") {
			c.UpdateMembership(chatID, userID, td.ChatMemberStatusLeft{})
			return td.ChatMemberStatusLeft{}, nil
		}

		return nil, fmt.Errorf("GetChatMember chat=%d user=%d: %w", chatID, userID, err)
	}

	c.UpdateMembership(chatID, userID, member.Status)
	return member.Status, nil
}

// joinUb joins the assistant to chatID via an invite link.
func (c *TelegramCalls) joinUb(chatID int64) error {
	call, err := c.GetGroupAssistant(chatID)
	if err != nil {
		return err
	}

	ub := call.App
	cacheKey := strconv.FormatInt(chatID, 10)

	link, err := c.resolveInviteLink(chatID, cacheKey)
	if err != nil {
		return err
	}

	logger.Info("joining via invite link", "chat_id", chatID)

	_, err = ub.JoinChannel(link)
	if err != nil {
		return c.handleJoinError(chatID, ub.Me().ID, err)
	}

	c.UpdateMembership(chatID, ub.Me().ID, td.ChatMemberStatusMember{})
	return nil
}

// resolveInviteLink returns a cached invite link or creates a new one.
func (c *TelegramCalls) resolveInviteLink(chatID int64, cacheKey string) (string, error) {
	if cached, ok := c.inviteCache.Get(cacheKey); ok && cached != "" {
		return cached, nil
	}

	chatLink, err := c.bot.CreateChatInviteLink(
		chatID, 0, 10, "FallenBeatz",
		&td.CreateChatInviteLinkOpts{CreatesJoinRequest: false},
	)
	if err != nil {
		return "", fmt.Errorf("create invite link for chat %d: %w", chatID, err)
	}

	link := chatLink.InviteLink
	if link == "" {
		return "", errors.New("telegram returned an empty invite link")
	}

	link = strings.Replace(link, "https://t.me/+", "https://t.me/joinchat/", 1)
	c.UpdateInviteLink(chatID, link)
	return link, nil
}

// handleJoinError maps JoinChannel error strings to actionable responses.
func (c *TelegramCalls) handleJoinError(chatID, userID int64, err error) error {
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "INVITE_REQUEST_SENT"):
		time.Sleep(time.Second)
		if approveErr := c.bot.ProcessChatJoinRequest(
			chatID, userID,
			&td.ProcessChatJoinRequestOpts{Approve: true},
		); approveErr != nil {
			slog.Warn("failed to approve join request", "error", approveErr)
			return fmt.Errorf("assistant (<code>%d</code>) has a pending join request", userID)
		}
		return nil

	case strings.Contains(errStr, "USER_ALREADY_PARTICIPANT"):
		c.UpdateMembership(chatID, userID, td.ChatMemberStatusMember{})
		return nil

	case strings.Contains(errStr, "INVITE_HASH_EXPIRED"):
		c.inviteCache.Delete(strconv.FormatInt(chatID, 10))
		return fmt.Errorf("invite link expired; assistant (<code>%d</code>) may be banned", userID)

	case strings.Contains(errStr, "CHANNEL_PRIVATE"):
		c.UpdateMembership(chatID, userID, td.ChatMemberStatusLeft{})
		c.inviteCache.Delete(strconv.FormatInt(chatID, 10))
		return fmt.Errorf("assistant (<code>%d</code>) is banned from this group", userID)
	}

	logger.Warn("unhandled JoinChannel error", "error", err)
	return err
}
