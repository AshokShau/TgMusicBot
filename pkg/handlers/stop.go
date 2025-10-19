package handlers

import (
	"fmt"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

// stopHandler handles the /stop command.
func stopHandler(m *telegram.NewMessage) error {
	chatId, _ := getPeerId(m.Client, m.ChatID())
	if !cache.ChatCache.IsActive(chatId) {
		_, _ = m.Reply("⏸ There is no track currently playing.")
		return nil
	}

	if err := vc.Calls.Stop(chatId); err != nil {
		_, _ = m.Reply("❌ An error occurred while stopping the playback: " + err.Error())
		return err
	}

	_, _ = m.Reply(fmt.Sprintf("⏹️ Playback has been stopped by %s, and the queue has been cleared.", m.Sender.FirstName))
	return nil
}
