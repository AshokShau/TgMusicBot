/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package vc

import (
	"fmt"
	"strings"
	"time"

	"ashokshau/tgmusic/src/core/cache"
	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

// LeaveAll makes the bot leave all groups and channels it's currently in,
// except for the ones specified in the exceptions list.
// Returns the number of chats left and any error encountered.
func (c *TelegramCalls) LeaveAll() (int, error) {
	leftCount := 0

	for _, call := range c.uBContext {
		userBot := call.App

		dialogs, err := userBot.GetDialogs(&telegram.DialogOptions{
			Limit:            -1,
			SleepThresholdMs: 20,
		})
		if err != nil {
			return leftCount, fmt.Errorf("failed to get dialogs: %w", err)
		}

		gologging.InfoF("for %s found %d dialogs", userBot.Me().FirstName, len(dialogs))
		activeChats := make(map[int64]bool)
		for _, id := range cache.ChatCache.GetActiveChats() {
			activeChats[id] = true
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
				gologging.WarnF("Unknown peer type: %T", peer)
				continue
			}

			if chatID == 0 {
				continue
			}

			// Skip if this is an active chat
			if activeChats[chatID] {
				continue
			}

			err = userBot.LeaveChannel(chatID)
			if err != nil {
				if strings.Contains(err.Error(), "USER_NOT_PARTICIPANT") || strings.Contains(err.Error(), "CHANNEL_PRIVATE") {
					continue
				}
				gologging.WarnF("Failed to leave chat %d: %v", chatID, err)
				continue
			}

			leftCount++
			time.Sleep(500 * time.Millisecond)
		}
	}

	return leftCount, nil
}
