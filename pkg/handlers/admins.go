package handlers

import (
	"fmt"
	"time"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"

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

	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	reloadKey := fmt.Sprintf("reload:%d", chatID)
	if lastUsed, ok := reloadRateLimit.Get(reloadKey); ok {
		timePassed := time.Since(lastUsed)
		if timePassed < reloadCooldown {
			remaining := int((reloadCooldown - timePassed).Seconds())
			_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "reload_cooldown"), cache.SecToMin(remaining)))
			return nil
		}
	}

	reloadRateLimit.Set(reloadKey, time.Now())
	reply, _ := m.Reply(lang.GetString(langCode, "reloading_admins"))

	cache.ClearAdminCache(chatID)
	admins, err := cache.GetAdmins(m.Client, chatID, true)
	if err != nil {
		gologging.WarnF("Failed to reload the admin cache for chat %d: %v", chatID, err)
		_, _ = reply.Edit(lang.GetString(langCode, "reload_error"))
		return nil
	}

	gologging.InfoF("Reloaded %d admins for chat %d", len(admins), chatID)
	_, err = reply.Edit(lang.GetString(langCode, "reload_success"))
	return err
}
