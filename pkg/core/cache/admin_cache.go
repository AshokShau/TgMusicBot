package cache

import (
	"fmt"
	"time"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

// AdminCache is a cache for chat administrators.
var AdminCache = NewCache[[]*telegram.Participant](time.Hour)

// GetChatAdmins retrieves the list of admin IDs for a given chat from the cache.
// It takes a chat ID and returns a slice of admin IDs, or an error if the admins are not found in the cache.
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
// It accepts a Telegram client, a chat ID, and a boolean to force a reload from the API, bypassing the cache.
// It returns a slice of telegram.Participant objects and any error encountered.
func GetAdmins(client *telegram.Client, chatID int64, forceReload bool) ([]*telegram.Participant, error) {
	cacheKey := fmt.Sprintf("admins:%d", chatID)
	if !forceReload {
		if admins, ok := AdminCache.Get(cacheKey); ok {
			return admins, nil
		}
	}

	opts := &telegram.ParticipantOptions{
		Filter:           &telegram.ChannelParticipantsAdmins{},
		SleepThresholdMs: 3000,
	}

	admins, _, err := client.GetChatMembers(chatID, opts)
	if err != nil {
		return nil, err
	}

	AdminCache.Set(cacheKey, admins)
	return admins, nil
}

// GetUserAdmin retrieves the participant information for a single administrator in a chat.
// It accepts a Telegram client, a chat ID, a user ID, and a boolean to force a reload from the API.
// It returns a telegram.Participant object or an error if the user is not an admin.
func GetUserAdmin(client *telegram.Client, chatID int64, userID int64, forceReload bool) (*telegram.Participant, error) {
	admins, err := GetAdmins(client, chatID, forceReload)
	if err != nil {
		gologging.WarnF("GetUserAdmin error: %v", err)
		// Cache a negative result for a short period to avoid repeated failed lookups.
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
// If the chatID is 0, it clears the entire admin cache. Otherwise, it clears the cache for a specific chat.
func ClearAdminCache(chatID int64) {
	if chatID == 0 {
		AdminCache.Clear()
		return
	}

	cacheKey := fmt.Sprintf("admins:%d", chatID)
	AdminCache.Delete(cacheKey)
}
