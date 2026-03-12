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

	td "github.com/AshokShau/gotdbot"
)

// AdminCache is the package-level cache for chat administrator lists.
// It is intentionally never Close()d because it lives for the process lifetime.
var AdminCache = NewCache[[]*td.ChatMember](time.Hour)

// allCreatorRights is the full permission set implicitly held by a chat creator.
var allCreatorRights = &td.ChatAdministratorRights{
	CanChangeInfo:           true,
	CanDeleteMessages:       true,
	CanDeleteStories:        true,
	CanEditMessages:         true,
	CanEditStories:          true,
	CanInviteUsers:          true,
	CanManageChat:           true,
	CanManageDirectMessages: true,
	CanManageTags:           true,
	CanManageTopics:         true,
	CanManageVideoChats:     true,
	CanPinMessages:          true,
	CanPostMessages:         true,
	CanPostStories:          true,
	CanPromoteMembers:       true,
	CanRestrictMembers:      true,
	IsAnonymous:             false,
}

// adminCacheKey returns the canonical cache key for a chat's admin list.
func adminCacheKey(chatID int64) string {
	return fmt.Sprintf("admins:%d", chatID)
}

// GetChatAdminIDs returns the user IDs of all cached admins for chatID.
// Returns an error if the chat is not in the cache (caller should use GetAdmins).
func GetChatAdminIDs(chatID int64) ([]int64, error) {
	admins, ok := AdminCache.Get(adminCacheKey(chatID))
	if !ok {
		return nil, fmt.Errorf("admins for chat %d not in cache", chatID)
	}

	ids := make([]int64, 0, len(admins))
	for _, admin := range admins {
		if user, ok := admin.MemberId.(*td.MessageSenderUser); ok {
			ids = append(ids, user.UserId)
		}
	}
	return ids, nil
}

// GetAdmins returns the administrator list for chatID.
// It serves from cache unless forceReload is true, in which case it always
// fetches from Telegram and refreshes the cache.
func GetAdmins(client *td.Client, chatID int64, forceReload bool) ([]*td.ChatMember, error) {
	key := adminCacheKey(chatID)

	if !forceReload {
		if admins, ok := AdminCache.Get(key); ok {
			return admins, nil
		}
	}

	res, err := client.SearchChatMembers(
		chatID,
		0,
		"",
		&td.SearchChatMembersOpts{
			Filter: td.ChatMembersFilterAdministrators{},
		},
	)
	if err != nil {
		// Do NOT cache the error — a transient failure should not block future
		// lookups for up to an hour. Let the next call retry.
		return nil, fmt.Errorf("fetch admins for chat %d: %w", chatID, err)
	}

	admins := make([]*td.ChatMember, len(res.Members))
	for i := range res.Members {
		admins[i] = &res.Members[i]
	}

	AdminCache.Set(key, admins)
	return admins, nil
}

// GetUserAdmin returns the ChatMember record for userID in chatID, or an error
// if they are not an administrator.
func GetUserAdmin(client *td.Client, chatID, userID int64, forceReload bool) (*td.ChatMember, error) {
	admins, err := GetAdmins(client, chatID, forceReload)
	if err != nil {
		return nil, err
	}

	for _, admin := range admins {
		if user, ok := admin.MemberId.(*td.MessageSenderUser); ok && user.UserId == userID {
			return admin, nil
		}
	}

	return nil, fmt.Errorf("user %d is not an administrator in chat %d", userID, chatID)
}

// GetRights returns the administrator rights for userID in chatID.
// Chat creators are granted the full permission set.
func GetRights(client *td.Client, chatID, userID int64, forceReload bool) (*td.ChatAdministratorRights, error) {
	admin, err := GetUserAdmin(client, chatID, userID, forceReload)
	if err != nil {
		return nil, err
	}

	switch status := admin.Status.(type) {
	case *td.ChatMemberStatusAdministrator:
		return status.Rights, nil
	case *td.ChatMemberStatusCreator:
		return allCreatorRights, nil
	default:
		// Unreachable in practice: GetUserAdmin only returns members whose
		// MemberId matched userID, and Telegram only lists admins/creators in
		// the administrators filter.
		return nil, fmt.Errorf("user %d has unexpected member status in chat %d", userID, chatID)
	}
}

// ClearAdminCache removes the cached admin list for chatID.
// Pass chatID 0 to clear all cached admin lists.
func ClearAdminCache(chatID int64) {
	if chatID == 0 {
		AdminCache.Clear()
		return
	}
	AdminCache.Delete(adminCacheKey(chatID))
}
