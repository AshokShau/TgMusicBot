package handlers

import (
	"fmt"

	"github.com/AshokShau/TgMusicBot/pkg/core"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

// pauseHandler handles the /pause command.
func pauseHandler(m *telegram.NewMessage) error {
	chatId, _ := getPeerId(m.Client, m.ChatID())

	if !cache.ChatCache.IsActive(chatId) {
		_, _ = m.Reply("⏸ There is no track currently playing.")
		return nil
	}

	if _, err := vc.Calls.Pause(chatId); err != nil {
		_, _ = m.Reply("❌ An error occurred while pausing the playback: " + err.Error())
		return nil
	}

	_, err := m.Reply(fmt.Sprintf("⏸️ Playback has been paused by %s.", m.Sender.FirstName), telegram.SendOptions{ReplyMarkup: core.ControlButtons("pause")})
	return err
}

// resumeHandler handles the /resume command.
func resumeHandler(m *telegram.NewMessage) error {
	chatId, _ := getPeerId(m.Client, m.ChatID())
	if chatId > 0 {
		_, _ = m.Reply("This command can only be used in a supergroup.")
		return nil
	}

	if !cache.ChatCache.IsActive(chatId) {
		_, _ = m.Reply("⏸ There is no track currently playing.")
		return nil
	}

	if _, err := vc.Calls.Resume(chatId); err != nil {
		_, _ = m.Reply("❌ An error occurred while resuming the playback: " + err.Error())
		return nil
	}

	_, err := m.Reply(fmt.Sprintf("▶️ Playback has been resumed by %s.", m.Sender.FirstName), telegram.SendOptions{ReplyMarkup: core.ControlButtons("resume")})
	return err
}
