/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package cache

import (
	"fmt"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

// AdminCache is a cache for chat administrators.
var AdminCache = NewCache[[]*telegram.Participant](time.Hour)

// GetChatAdmins retrieves the list of admin IDs for a given chat from the cache.
func GetChatAdmins(chatID int64) ([]int64, error) {
	cacheKey := fmt.Sprintf("admins:%d", chatID)
	if admins, ok := AdminCache.Get(cacheKey); ok {
		var adminIDs []int64
		for _, admin := range admins {
			adminIDs = append(adminIDs, admin.User.ID)
		}
		return adminIDs, nil
	}
	return nil, fmt.Errorf("could not find admins in cache for chat %d", chatID)
}

// GetAdmins fetches a list of administrators from the cache or, if not present, from the Telegram API.
func GetAdmins(client *telegram.Client, chatID int64, forceReload bool) ([]*telegram.Participant, error) {
	cacheKey := fmt.Sprintf("admins:%d", chatID)
	if !forceReload {
		if admins, ok := AdminCache.Get(cacheKey); ok {
			return admins, nil
		}
	}

	opts := &telegram.ParticipantOptions{
		Filter: &telegram.ChannelParticipantsAdmins{},
		Limit:  -1,
	}

	admins, _, err := client.GetChatMembers(chatID, opts)
	if err != nil {
		client.Logger.Warn("GetAdmins error: %v", err)
		return nil, err
	}

	AdminCache.Set(cacheKey, admins)
	return admins, nil
}

// GetUserAdmin retrieves the participant information for a single administrator in a chat.
func GetUserAdmin(client *telegram.Client, chatID, userID int64, forceReload bool) (*telegram.Participant, error) {
	admins, err := GetAdmins(client, chatID, forceReload)
	if err != nil {
		client.Logger.Warn("GetUserAdmin error: %v", err)
		cacheKey := fmt.Sprintf("admins:%d", chatID)
		AdminCache.SetWithTTL(cacheKey, []*telegram.Participant{}, 10*time.Minute)
		return nil, err
	}

	for _, admin := range admins {
		if admin.User.ID == userID {
			return admin, nil
		}
	}

	return nil, fmt.Errorf("user %d is not an administrator in chat %d", userID, chatID)
}

// ClearAdminCache removes cached administrator lists.
func ClearAdminCache(chatID int64) {
	if chatID == 0 {
		AdminCache.Clear()
		return
	}

	cacheKey := fmt.Sprintf("admins:%d", chatID)
	AdminCache.Delete(cacheKey)
}
