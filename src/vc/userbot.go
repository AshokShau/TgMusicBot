/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package vc

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/vc/sessions"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"ashokshau/tgmusic/src/core/cache"

	td "github.com/AshokShau/gotdbot"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// joinAssistant ensures the assistant is a member of the specified chat.
func (c *TelegramCalls) joinAssistant(bot *td.Client, chatID int64, call *Assistant, index int) error {
	status, err := c.checkUserStats(bot, chatID, call, index)
	if err != nil {
		return fmt.Errorf("(client%d): failed to check user status: %w", index, err)
	}

	logger.Info("chat member status", "chat_id", chatID, "status", status, "index", index)

	switch status.(type) {
	case *td.ChatMemberStatusMember, td.ChatMemberStatusCreator, td.ChatMemberStatusAdministrator, td.ChatMemberStatusMember, *td.ChatMemberStatusAdministrator, *td.ChatMemberStatusCreator:
		return nil

	case *td.ChatMemberStatusLeft, td.ChatMemberStatusLeft:
		logger.Info("assistant is not in chat, joining", "chat_id", chatID, "index", index)
		return c.joinUb(bot, chatID, call, index)

	case *td.ChatMemberStatusBanned, *td.ChatMemberStatusRestricted,
		td.ChatMemberStatusBanned, td.ChatMemberStatusRestricted:
		_, isBannedPtr := status.(*td.ChatMemberStatusBanned)
		_, isBannedVal := status.(td.ChatMemberStatusBanned)
		isBanned := isBannedPtr || isBannedVal

		_, isMutedPtr := status.(*td.ChatMemberStatusRestricted)
		_, isMutedVal := status.(td.ChatMemberStatusRestricted)
		isMuted := isMutedPtr || isMutedVal

		logger.Info("assistant is banned or restricted, attempting recovery",
			"chat_id", chatID, "banned", isBanned, "muted", isMuted, "index", index)

		return c.recoverBannedAssistant(bot, chatID, call, index, isBanned)

	default:
		logger.Warn("unknown assistant status, attempting to join", "status", status, "index", index)
		return c.joinUb(bot, chatID, call, index)
	}
}

// recoverBannedAssistant attempts to unban or unmute the assistant using bot admin rights.
func (c *TelegramCalls) recoverBannedAssistant(bot *td.Client, chatID int64, call *Assistant, index int, isBanned bool) error {
	ubID := call.App.Me().ID
	botStatus, err := cache.GetUserAdmin(bot, chatID, bot.Me.Id, false)
	if err != nil {
		if strings.Contains(err.Error(), "is not an admin in chat") {
			return fmt.Errorf(
				"client%d: bot is not an admin, cannot unban my assistant (<code>%d</code>)",
				index, ubID,
			)
		}
		return fmt.Errorf("failed to check bot admin status: %w", err)
	}

	admin, ok := botStatus.Status.(*td.ChatMemberStatusAdministrator)
	if !ok || admin.Rights == nil || !admin.Rights.CanRestrictMembers {
		return fmt.Errorf(
			"client%d is banned in your group (<code>%d</code>) & bot lacks CanRestrictMembers to unban my assistant",
			index, ubID,
		)
	}

	if isBanned {
		if err := bot.SetChatMemberStatus(
			chatID,
			td.MessageSenderUser{UserId: ubID},
			&td.ChatMemberStatusMember{},
		); err != nil {
			logger.Warn("failed to unban assistant", "ub_id", ubID, "error", err, "index", index)
		}

		return c.joinUb(bot, chatID, call, index)
	}

	// isMuted: restricted but not banned — nothing actionable right now.
	// TODO: call SetChatMemberStatus to lift restrictions.
	return nil
}

// clientIndexFor returns the 0-based index for the given call, or -1 if not found.
// Caller must not hold mu.
func (c *TelegramCalls) clientIndexFor(call *Assistant) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for idx, ctx := range c.assistants {
		if ctx == call {
			return idx
		}
	}
	return -1
}

// checkUserStats returns the assistant's membership status in chatID.
// Results are cached; a cache miss triggers a live Telegram API call.
func (c *TelegramCalls) checkUserStats(bot *td.Client, chatID int64, call *Assistant, index int) (td.ChatMemberStatus, error) {
	userID := call.App.Me().ID
	cacheKey := fmt.Sprintf("%d:%d", chatID, userID)
	if cached, ok := c.statusCache.Get(cacheKey); ok {
		return cached, nil
	}

	member, err := bot.GetChatMember(chatID, td.MessageSenderUser{UserId: userID})
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "USER_NOT_PARTICIPANT") {
			c.UpdateMembership(chatID, userID, &td.ChatMemberStatusLeft{})
			return &td.ChatMemberStatusLeft{}, nil
		}

		return nil, fmt.Errorf("GetChatMember (client %d) chat=%d user=%d: %w", index, chatID, userID, err)
	}

	c.UpdateMembership(chatID, userID, member.Status)
	return member.Status, nil
}

// joinUb joins the assistant to chatID via an ChatInviteLink link.
func (c *TelegramCalls) joinUb(bot *td.Client, chatID int64, call *Assistant, index int) error {
	ub := call.App
	cacheKey := strconv.FormatInt(chatID, 10)

	link, err := c.resolveInviteLink(bot, chatID, cacheKey)
	if err != nil {
		return err
	}

	logger.Info("joining via invite link", "chat_id", chatID, "index", index)

	_, err = ub.JoinChannel(link)
	if err != nil {
		return c.handleJoinError(bot, chatID, ub.Me().ID, index, err)
	}

	c.UpdateMembership(chatID, ub.Me().ID, &td.ChatMemberStatusMember{})
	return nil
}

// resolveInviteLink returns a cached invite link or creates a new one.
func (c *TelegramCalls) resolveInviteLink(bot *td.Client, chatID int64, cacheKey string) (string, error) {
	if cached, ok := c.inviteCache.Get(cacheKey); ok && cached != "" {
		return cached, nil
	}

	chatLink, err := bot.CreateChatInviteLink(
		chatID, 0, 0, "FallenBeatz",
		&td.CreateChatInviteLinkOpts{CreatesJoinRequest: false},
	)

	if err != nil {
		return "", fmt.Errorf("create invite link for chat %d: %w", chatID, err)
	}

	link := chatLink.InviteLink
	if link == "" {
		return "", errors.New("telegram returned an empty invite link")
	}

	c.UpdateInviteLink(chatID, link)
	return link, nil
}

// handleJoinError maps JoinChannel error strings to actionable responses.
func (c *TelegramCalls) handleJoinError(bot *td.Client, chatID, userID int64, index int, err error) error {
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "INVITE_REQUEST_SENT"):
		time.Sleep(time.Second)
		if approveErr := bot.ProcessChatJoinRequest(
			chatID, userID,
			&td.ProcessChatJoinRequestOpts{Approve: true},
		); approveErr != nil {
			slog.Warn("failed to approve join request", "error", approveErr, "index", index)
			return fmt.Errorf("client %d: assistant (<code>%d</code>) has a pending join request: %v", index, userID, approveErr)
		}
		return nil

	case strings.Contains(errMsg, "USER_ALREADY_PARTICIPANT"):
		c.UpdateMembership(chatID, userID, &td.ChatMemberStatusMember{})
		return nil

	case strings.Contains(errMsg, "INVITE_HASH_EXPIRED"):
		cached, _ := c.inviteCache.Get(strconv.FormatInt(chatID, 10))
		logger.Warn("invite link expired", "chat_id", chatID, "index", index, "cached_link", cached)
		c.inviteCache.Delete(strconv.FormatInt(chatID, 10))
		c.UpdateMembership(chatID, userID, &td.ChatMemberStatusLeft{})
		return fmt.Errorf("client %d: assistant (<code>%d</code>) invite link expired", index, userID)

	case strings.Contains(errMsg, "CHANNEL_PRIVATE"):
		c.inviteCache.Delete(strconv.FormatInt(chatID, 10))
		c.UpdateMembership(chatID, userID, &td.ChatMemberStatusLeft{})
		return fmt.Errorf("client %d: assistant (<code>%d</code>) is banned from this group", index, userID)
	}

	logger.Warn("unhandled JoinChannel error", "error", err, "index", index)
	return fmt.Errorf("(client%d, <code>%d</code>): assistant failed to join: %w", index, userID, err)
}

func (c *TelegramCalls) StartClient(apiID int32, apiHash, stringSession string) (*Assistant, error) {
	c.mu.Lock()
	clientIndex := len(c.assistants)
	c.mu.Unlock()

	clientName := fmt.Sprintf("client%d", clientIndex)

	var sess *tg.Session
	var err error

	clientConfig := tg.ClientConfig{
		AppID:         apiID,
		AppHash:       apiHash,
		MemorySession: true,
		SessionName:   clientName,
		FloodHandler:  handleFlood,
		LogLevel:      tg.InfoLevel,
	}

	switch config.SessionType {
	case "telethon":
		sess, err = sessions.DecodeTelethonSessionString(stringSession)
		if err != nil {
			return nil, fmt.Errorf("failed to decode telethon session string for %s: %w", clientName, err)
		}
		clientConfig.StringSession = sess.Encode()
	case "pyrogram":
		sess, err = sessions.DecodePyrogramSessionString(stringSession)
		if err != nil {
			return nil, fmt.Errorf("failed to decode pyrogram session string for %s: %w", clientName, err)
		}
		clientConfig.StringSession = sess.Encode()
	case "gogram":
		clientConfig.StringSession = stringSession
	default:
		return nil, fmt.Errorf("unsupported session type: %s", config.SessionType)
	}

	mtProto, err := tg.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create the MTProto client: %w", err)
	}

	if err = mtProto.Start(); err != nil {
		return nil, fmt.Errorf("failed to start the client: %w", err)
	}

	me := mtProto.Me()
	if me.Bot {
		_ = mtProto.Stop()
		return nil, fmt.Errorf("the client %s is a bot", clientName)
	}

	appConfig, err := mtProto.HelpGetAppConfig(0)
	if err != nil {
		logger.Warn("[TelegramCalls] failed to fetch app config", "client", clientName, "error", err)
	} else {
		isFreeze := false
		if cfg, ok := appConfig.(*tg.HelpAppConfigObj); ok {
			if cfgObj, ok := cfg.Config.(*tg.JsonObject); ok {
				for _, entry := range cfgObj.Value {
					if entry != nil && entry.Key == "freeze_since_date" {
						isFreeze = true
						break
					}
				}
			}
		}

		if isFreeze {
			logger.Warn("[TelegramCalls] The client is frozen and cannot be used for voice calls", "client", clientName, "id", me.ID, "username", me.Username)
			_ = mtProto.Stop()
			return nil, nil
		}
	}

	call, err := newAssistant(mtProto)
	if err != nil {
		_ = mtProto.Stop()
		return nil, fmt.Errorf("failed to create the assistant instance: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.assistants[clientIndex] = call
	c.clients[clientIndex] = mtProto

	logger.Info("[TelegramCalls] Client started", "client", clientName, "id", me.ID, "username", me.Username)
	return call, nil
}

func (c *TelegramCalls) StopAllClients() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, call := range c.assistants {
		call.Close()
	}

	for idx, client := range c.clients {
		slog.Info("[TelegramCalls] Stopping the client", "index", idx)
		_ = client.Stop()
	}
}
