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
		return
	}

	go func() {
		// Initial delay to ensure clients and logger are fully initialized
		time.Sleep(30 * time.Second)

		if logger != nil {
			logger.Infof("Starting auto leave service - userbot will leave chats with no queue and no music playing for 10 minutes")
		}

		ticker := time.NewTicker(3 * time.Minute) // Check every 3 minutes
		defer ticker.Stop()

		for range ticker.C {
			c.checkInactiveChats()
		}
	}()
}

// checkInactiveChats checks for inactive chats and leaves them
func (c *TelegramCalls) checkInactiveChats() {
	c.mu.RLock()
	if len(c.uBContext) == 0 {
		c.mu.RUnlock()
		if logger != nil {
			logger.Warn("No active userbot clients available for auto leave check")
		}
		return
	}
	c.mu.RUnlock()

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
				logger.Warn("Failed to get dialogs for auto leave: %v", err)
			}
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
				// Channel IDs need -100 prefix for proper format
				chatID = int64(-1000000000000) - p.ChannelID
			case *telegram.PeerChat:
				// Chat IDs need - prefix
				chatID = -p.ChatID
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

			// Check if there's any queue in the chat
			queueLength := cache.ChatCache.GetQueueLength(chatID)
			if queueLength > 0 {
				// There's a queue, don't leave
				continue
			}

			// Check last active time
			lastActive, exists := activeChats[chatID]
			if !exists {
				// Never been active (joined but never played music)
				// This means userbot joined but never played music - should leave immediately
				if logger != nil {
					logger.Infof("Leaving chat %d (userbot joined but never played music)", chatID)
				}

				err = userBot.LeaveChannel(chatID)
				if err != nil {
					if strings.Contains(err.Error(), "USER_NOT_PARTICIPANT") ||
						strings.Contains(err.Error(), "CHANNEL_PRIVATE") {
						continue
					}
					if logger != nil {
						logger.Warn("Failed to leave inactive chat %d: %v", chatID, err)
					}
					continue
				}

				time.Sleep(500 * time.Millisecond)
				continue
			}

			// Calculate inactive duration (10 minutes = 600 seconds)
			inactiveDuration := time.Since(lastActive)
			inactiveThreshold := 10 * time.Minute // 10 menit

			if inactiveDuration >= inactiveThreshold {
				if logger != nil {
					logger.Infof("Leaving inactive chat %d (no queue, no playing music for %v)", chatID, inactiveDuration)
				}

				err = userBot.LeaveChannel(chatID)
				if err != nil {
					if strings.Contains(err.Error(), "USER_NOT_PARTICIPANT") ||
						strings.Contains(err.Error(), "CHANNEL_PRIVATE") {
						continue
					}
					if logger != nil {
						logger.Warn("Failed to leave inactive chat %d: %v", chatID, err)
					}
					continue
				}

				// Clear cache for this chat
				cache.ChatCache.ClearChat(chatID)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}
