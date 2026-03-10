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

// AdminCache is a cache for chat administrators.
var AdminCache = NewCache[[]*td.ChatMember](time.Hour)

// GetChatAdmins retrieves the list of admin IDs for a given chat from the cache.
func GetChatAdmins(chatID int64) ([]int64, error) {
	cacheKey := fmt.Sprintf("admins:%d", chatID)

	if admins, ok := AdminCache.Get(cacheKey); ok {
		var adminIDs []int64

		for _, admin := range admins {
			if user, ok := admin.MemberId.(*td.MessageSenderUser); ok {
				adminIDs = append(adminIDs, user.UserId)
			}
		}

		return adminIDs, nil
	}

	return nil, fmt.Errorf("could not find admins in cache for chat %d", chatID)
}

// GetAdmins fetches a list of administrators from the cache or from Telegram.
func GetAdmins(client *td.Client, chatID int64, forceReload bool) ([]*td.ChatMember, error) {
	cacheKey := fmt.Sprintf("admins:%d", chatID)

	if !forceReload {
		if admins, ok := AdminCache.Get(cacheKey); ok {
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
		return nil, err
	}

	admins := make([]*td.ChatMember, 0, len(res.Members))
	for i := range res.Members {
		member := res.Members[i]
		admins = append(admins, &member)
	}

	AdminCache.Set(cacheKey, admins)

	return admins, nil
}

// GetUserAdmin retrieves a specific administrator.
func GetUserAdmin(client *td.Client, chatID, userID int64, forceReload bool) (*td.ChatMember, error) {
	admins, err := GetAdmins(client, chatID, forceReload)

	if err != nil {
		cacheKey := fmt.Sprintf("admins:%d", chatID)
		AdminCache.SetWithTTL(cacheKey, []*td.ChatMember{}, 10*time.Minute)
		return nil, err
	}

	for _, admin := range admins {
		if user, ok := admin.MemberId.(*td.MessageSenderUser); ok {
			if user.UserId == userID {
				return admin, nil
			}
		}
	}

	return nil, fmt.Errorf("user %d is not an administrator in chat %d", userID, chatID)
}

func GetRights(client *td.Client, chatID, userID int64, forceReload bool) (*td.ChatAdministratorRights, error) {
	admin, err := GetUserAdmin(client, chatID, userID, forceReload)
	if err != nil {
		return nil, err
	}

	switch status := admin.Status.(type) {
	case *td.ChatMemberStatusAdministrator:
		return status.Rights, nil

	case *td.ChatMemberStatusCreator:
		// creator implicitly has all permissions
		return &td.ChatAdministratorRights{
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
		}, nil
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
