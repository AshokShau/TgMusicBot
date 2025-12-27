/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package vc

import (
	"strings"
	"time"

	"ashokshau/tgmusic/src/config"

	"github.com/amarnathcjd/gogram/telegram"
)

// LeaveAllOnStartup leaves all chats except logger group when userbot starts
// This frees up join slots so userbot can join new groups
func (c *TelegramCalls) LeaveAllOnStartup() {
	c.mu.RLock()
	if len(c.uBContext) == 0 {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	// Wait a bit for clients to be fully ready
	time.Sleep(5 * time.Second)

	if logger != nil {
		logger.Infof("Starting startup leave-all - userbot will leave all chats except logger group")
	}

	leftCount := 0
	loggerID := config.Conf.LoggerId

	for _, call := range c.uBContext {
		if call == nil || call.App == nil {
			continue
		}

		userBot := call.App

		dialogs, err := userBot.GetDialogs(&telegram.DialogOptions{
			Limit:            -1,
			SleepThresholdMs: 20,
		})
		if err != nil {
			if logger != nil {
				logger.Warn("Failed to get dialogs for startup leave: %v", err)
			}
			continue
		}

		if logger != nil {
			logger.Infof("Found %d dialogs for %s", len(dialogs), userBot.Me().FirstName)
		}

		for _, d := range dialogs {
			peer := d.Peer
			var chatID int64
			switch p := peer.(type) {
			case *telegram.PeerChannel:
				chatID = p.ChannelID
			case *telegram.PeerChat:
				chatID = p.ChatID
			case *telegram.PeerUser:
				continue // Skip private chats
			default:
				continue
			}

			if chatID == 0 {
				continue
			}

			// Skip logger group
			if chatID == loggerID || chatID == -loggerID || chatID == -100-loggerID {
				if logger != nil {
					logger.Infof("Skipping logger group %d on startup leave", chatID)
				}
				continue
			}

			// Also check with -100 prefix for supergroups
			if loggerID > 0 && (chatID == -100*1000000000-loggerID || -chatID == loggerID) {
				if logger != nil {
					logger.Infof("Skipping logger group %d on startup leave", chatID)
				}
				continue
			}

			err = userBot.LeaveChannel(chatID)
			if err != nil {
				if strings.Contains(err.Error(), "USER_NOT_PARTICIPANT") ||
					strings.Contains(err.Error(), "CHANNEL_PRIVATE") {
					continue
				}
				if logger != nil {
					logger.Warn("Failed to leave chat %d on startup: %v", chatID, err)
				}
				continue
			}

			leftCount++
			if logger != nil {
				logger.Infof("Left chat %d on startup", chatID)
			}

			time.Sleep(500 * time.Millisecond) // Avoid flood
		}
	}

	if logger != nil {
		logger.Infof("Startup leave-all completed. Left %d chats.", leftCount)
	}
}
