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
	"ashokshau/tgmusic/src/core/cache"

	"github.com/amarnathcjd/gogram/telegram"
)

// StartAutoLeaveService starts the auto leave service for inactive chats
func (c *TelegramCalls) StartAutoLeaveService() {
	if config.Conf.AutoLeaveTime <= 0 {
		logger.Info("Auto leave service is disabled (AUTO_LEAVE_TIME not set or <= 0)")
		return
	}

	logger.Infof("Starting auto leave service with timeout: %d seconds", config.Conf.AutoLeaveTime)

	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
		defer ticker.Stop()

		for range ticker.C {
			c.checkInactiveChats()
		}
	}()
}

// checkInactiveChats checks for inactive chats and leaves them
func (c *TelegramCalls) checkInactiveChats() {
	for _, call := range c.uBContext {
		userBot := call.App

		dialogs, err := userBot.GetDialogs(&telegram.DialogOptions{
			Limit:            -1,
			SleepThresholdMs: 20,
		})
		if err != nil {
			logger.Warn("Failed to get dialogs for auto leave: %v", err)
			continue
		}

		activeChats := make(map[int64]time.Time)
		for chatID, lastActive := range cache.ChatCache.GetLastActiveTimes() {
			activeChats[chatID] = lastActive
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
				continue
			default:
				continue
			}

			if chatID == 0 {
				continue
			}

			// Check if chat is currently active (playing music)
			if cache.ChatCache.IsActive(chatID) {
				continue
			}

			// Check last active time
			lastActive, exists := activeChats[chatID]
			if !exists {
				// Never been active, skip
				continue
			}

			// Calculate inactive duration
			inactiveDuration := time.Since(lastActive)
			if inactiveDuration.Seconds() >= float64(config.Conf.AutoLeaveTime) {
				logger.Infof("Leaving inactive chat %d (inactive for %v)", chatID, inactiveDuration)

				err = userBot.LeaveChannel(chatID)
				if err != nil {
					if strings.Contains(err.Error(), "USER_NOT_PARTICIPANT") || 
					   strings.Contains(err.Error(), "CHANNEL_PRIVATE") {
						continue
					}
					logger.Warn("Failed to leave inactive chat %d: %v", chatID, err)
					continue
				}

				// Clear cache for this chat
				cache.ChatCache.ClearChat(chatID)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}
