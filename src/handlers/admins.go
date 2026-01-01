/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/utils"
	"ashokshau/tgmusic/src/vc"
	"fmt"
	"time"

	"ashokshau/tgmusic/src/core/cache"

	tg "github.com/amarnathcjd/gogram/telegram"
)

const reloadCooldown = 3 * time.Minute

var reloadRateLimit = cache.NewCache[time.Time](reloadCooldown)

// reloadAdminCacheHandler reloads the admin cache for a chat.
func reloadAdminCacheHandler(m *tg.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	chatID := m.ChannelID()
	reloadKey := fmt.Sprintf("reload:%d", chatID)

	if lastUsed, ok := reloadRateLimit.Get(reloadKey); ok {
		timePassed := time.Since(lastUsed)
		if timePassed < reloadCooldown {
			remaining := int((reloadCooldown - timePassed).Seconds())
			_, _ = m.Reply(fmt.Sprintf("⏳ Please wait %s before using this command again.", utils.SecToMin(remaining)))
			return nil
		}
	}

	reloadRateLimit.Set(reloadKey, time.Now())
	reply, err := m.Reply("🔄 Reloading admin cache...")
	if err != nil {
		logger.Warn("Failed to send reloading message for chat %d: %v", chatID, err)
		return tg.ErrEndGroup
	}

	cache.ClearAdminCache(chatID)
	// cache.ChatCache.ClearChat(chatID)
	vc.Calls.UpdateInviteLink(chatID, "")
	admins, err := cache.GetAdmins(m.Client, chatID, true)
	if err != nil {
		logger.Warn("Failed to reload the admin cache for chat %d: %v", chatID, err)
		_, _ = reply.Edit("⚠️ Error reloading admin cache.")
		return nil
	}

	logger.Info("Reloaded %d admins for chat %d", len(admins), chatID)
	_, _ = reply.Edit("✅ Admin cache reloaded.")
	return nil
}
