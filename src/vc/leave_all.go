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
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/vc/ubot"

	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

func (c *TelegramCalls) LeaveAll() (int, error) {
	var totalLeft atomic.Int64
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	c.mu.RLock()
	var ubContexts []*ubot.Context
	for _, call := range c.uBContext {
		ubContexts = append(ubContexts, call)
	}
	c.mu.RUnlock()

	for _, call := range ubContexts {
		wg.Add(1)
		go func(ctx *ubot.Context) {
			defer wg.Done()
			count, err := c.leaveAssistantDialogs(ctx)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}
			totalLeft.Add(int64(count))
		}(call)
	}

	wg.Wait()
	return int(totalLeft.Load()), firstErr
}

func (c *TelegramCalls) LeaveAllForClient(index int) (int, error) {
	c.mu.RLock()
	call, ok := c.uBContext[index]
	c.mu.RUnlock()
	if !ok {
		return 0, fmt.Errorf("no ntgcalls instance was found for client index %d", index)
	}
	return c.leaveAssistantDialogs(call)
}

func (c *TelegramCalls) leaveAssistantDialogs(ctx *ubot.Context) (int, error) {
	userBot := ctx.App
	var totalLeft int
	dialogs, err := userBot.GetDialogs(&telegram.DialogOptions{
		Limit:            -1,
		SleepThresholdMs: 20,
	})
	if err != nil {
		return 0, fmt.Errorf("account %s: failed to get dialogs: %w",
			userBot.Me().FirstName, err)
	}

	logger.Info("found dialogs",
		"user", userBot.Me().FirstName,
		"count", len(dialogs),
	)

	for _, d := range dialogs {
		var chatID int64
		switch p := d.Peer.(type) {
		case *telegram.PeerChannel:
			chatID = p.ChannelID
		case *telegram.PeerChat:
			chatID = p.ChatID
		default:
			continue
		}

		if chatID == 0 {
			continue
		}

		if cache.ChatCache.IsActive(chatID) {
			continue
		}

		for {
			if cache.ChatCache.IsActive(chatID) {
				break
			}

			err = userBot.LeaveChannel(chatID)
			if err == nil {
				totalLeft++
				break
			}
			if strings.Contains(err.Error(), "USER_NOT_PARTICIPANT") ||
				strings.Contains(err.Error(), "CHANNEL_PRIVATE") {
				break
			}
			wait := telegram.GetFloodWait(err)
			if wait > 0 {
				logger.Warn("flood wait",
					"user", userBot.Me().FirstName,
					"chat_id", chatID,
					"seconds", wait,
				)
				time.Sleep(time.Duration(wait+20) * time.Second)
				continue
			}
			logger.Warn("leave failed",
				"user", userBot.Me().FirstName,
				"chat_id", chatID,
				"error", err,
			)
			break
		}

		time.Sleep(1 * time.Second)
	}
	return totalLeft, nil
}

const autoLeaveInterval = 18 * time.Hour

func (c *TelegramCalls) startAutoLeave(ctx context.Context) {
	if !config.Conf.AutoLeave {
		return
	}
	go func() {
		logger.Info("AutoLeave enabled, starting background task",
			"interval", autoLeaveInterval)
		ticker := time.NewTicker(autoLeaveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info("AutoLeave: background task stopped")
				return
			case <-ticker.C:
				c.runAutoLeave()
			}
		}
	}()
}

func (c *TelegramCalls) runAutoLeave() {
	logger.Info("AutoLeave: leaving inactive chats")
	leftCount, err := c.LeaveAll()
	if err != nil {
		logger.Error("AutoLeave: failed to leave chats", "error", err)
		return
	}
	logger.Info("AutoLeave: completed", "leftCount", leftCount)
	if leftCount > 0 && config.Conf.LoggerId != 0 {
		msg := fmt.Sprintf("AutoLeave: Assistant left %d inactive chats", leftCount)
		if _, err = c.bot.SendTextMessage(config.Conf.LoggerId, msg, nil); err != nil {
			logger.Error("AutoLeave: failed to send log message", "error", err)
		}
	}
}
