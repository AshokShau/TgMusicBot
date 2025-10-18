package handlers

import (
	"tgmusic/pkg/core/cache"
	"tgmusic/pkg/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

// skipHandler handles the /skip command.
func skipHandler(m *telegram.NewMessage) error {
	chatId, _ := getPeerId(m.Client, m.ChatID())

	if !cache.ChatCache.IsActive(chatId) {
		_, _ = m.Reply("⏸ There is no track currently playing.")
		return nil
	}

	_ = vc.Calls.PlayNext(chatId)
	return nil
}
