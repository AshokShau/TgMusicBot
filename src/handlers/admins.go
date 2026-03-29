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

	"github.com/AshokShau/gotdbot"
)

const reloadCooldown = 3 * time.Minute

var reloadRateLimit = cache.NewCache[time.Time](reloadCooldown)

// reloadAdminCacheHandler reloads the admin cache for a chat.
func reloadAdminCacheHandler(c *gotdbot.Client, ctx *gotdbot.Context) error {
	m := ctx.EffectiveMessage
	if m.IsPrivate() {
		return gotdbot.EndGroups
	}

	reloadKey := fmt.Sprintf("reload:%d", m.ChatId)
	if lastUsed, ok := reloadRateLimit.Get(reloadKey); ok {
		timePassed := time.Since(lastUsed)
		if timePassed < reloadCooldown {
			remaining := int((reloadCooldown - timePassed).Seconds())
			_, _ = m.ReplyText(c, fmt.Sprintf("⏳ Please wait %s before using this command again.", utils.SecToMin(remaining)), nil)
			return nil
		}
	}

	reloadRateLimit.Set(reloadKey, time.Now())
	reply, err := m.ReplyText(c, "🔄 Reloading admin cache...", nil)
	if err != nil {
		c.Logger.Warn("Failed to send reloading message for chat", "chat_id", m.ChatId, "error", err)
		return gotdbot.EndGroups
	}

	cache.ClearAdminCache(m.ChatId)
	// cache.ChatCache.ClearChat(m.ChatId)
	vc.Calls.UpdateInviteLink(m.ChatId, "")
	admins, err := cache.GetAdmins(c, m.ChatId, true)
	if err != nil {
		c.Logger.Warn("Failed to reload the admin cache for chat", "chat_id", m.ChatId, "error", err)
		_, _ = reply.EditText(c, "⚠️ Error reloading admin cache.", nil)
		return gotdbot.EndGroups
	}

	c.Logger.Info("Reloaded  admins for chat", "arg1", len(admins), "chat_id", m.ChatId)
	_, _ = reply.EditText(c, "✅ Admin cache reloaded.", nil)
	return gotdbot.EndGroups
}

// privacyHandler handles the /privacy command.
func privacyHandler(c *gotdbot.Client, ctx *gotdbot.Context) error {
	m := ctx.EffectiveMessage
	botName := c.Me.FirstName

	text := fmt.Sprintf("<b>Privacy Policy for %s</b>\n\n<b>1. Data Storage:</b>\nWe do not store personal data on your device. We do not track your browsing activity.\n\n<b>2. Collection:</b>\nWe only collect your Telegram <b>User ID</b> and <b>Chat ID</b> to provide music services. No names, phone numbers, or locations are stored.\n\n<b>3. Usage:</b>\nData is used strictly for bot functionality. No marketing or commercial use.\n\n<b>4. Sharing:</b>\nWe do not share data with third parties. No data is sold or traded.\n\n<b>5. Security:</b>\nWe use standard encryption to protect data. However, no online service is 100%% secure.\n\n<b>6. Cookies:</b>\n%s does not use cookies or tracking technologies.\n\n<b>7. Third Parties:</b>\nWe do not integrate with third-party data collectors, other than Telegram itself.\n\n<b>8. Your Rights:</b>\nYou can request data deletion or block the bot to revoke access.\n\n<b>9. Updates:</b>\nPolicy changes will be announced in the bot.\n\n<b>10. Contact:</b>\nQuestions? Contact our <a href=\"https://t.me/GuardxSupport\">Support Group</a>.\n\n──────────────────\n<b>Note:</b> This policy ensures a safe and respectful experience with %s.", botName, botName, botName)

	_, err := m.ReplyText(c, text, &gotdbot.SendTextMessageOpts{ParseMode: "html", DisableWebPagePreview: true})
	return err
}
