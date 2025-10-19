package handlers

import (
	"fmt"
	"time"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

const reloadCooldown = 3 * time.Minute

var reloadRateLimit = cache.NewCache[time.Time](reloadCooldown)

// reloadAdminCacheHandler reloads the admin cache for a chat.
func reloadAdminCacheHandler(m *telegram.NewMessage) error {
	if m.IsPrivate() {
		return nil
	}

	chatId, _ := getPeerId(m.Client, m.ChatID())

	reloadKey := fmt.Sprintf("reload:%d", chatId)
	if lastUsed, ok := reloadRateLimit.Get(reloadKey); ok {
		timePassed := time.Since(lastUsed)
		if timePassed < reloadCooldown {
			remaining := int((reloadCooldown - timePassed).Seconds())
			_, _ = m.Reply(fmt.Sprintf("⏳ Please wait %s before using this command again.", cache.SecToMin(remaining)))
			return nil
		}
	}

	reloadRateLimit.Set(reloadKey, time.Now())
	reply, _ := m.Reply("🔄 Reloading the admin cache...")

	cache.ClearAdminCache(chatId)
	admins, err := cache.GetAdmins(m.Client, chatId, true)
	if err != nil {
		gologging.WarnF("Failed to reload the admin cache for chat %d: %v", chatId, err)
		_, _ = reply.Edit("⚠️ An error occurred while reloading the admin cache.")
		return nil
	}

	gologging.InfoF("Reloaded %d admins for chat %d", len(admins), chatId)
	_, err = reply.Edit("✅ The admin cache has been successfully reloaded.")
	return err
}
