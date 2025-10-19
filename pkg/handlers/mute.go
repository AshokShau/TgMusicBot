package handlers

import (
	"fmt"

	"github.com/AshokShau/TgMusicBot/pkg/core"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

// muteHandler handles the /mute command.
func muteHandler(m *telegram.NewMessage) error {
	chatId, _ := getPeerId(m.Client, m.ChatID())

	if !cache.ChatCache.IsActive(chatId) {
		_, err := m.Reply("⏸ There is no track currently playing.")
		return err
	}

	if _, err := vc.Calls.Mute(chatId); err != nil {
		_, err = m.Reply("❌ An error occurred while muting the playback: " + err.Error())
		return err
	}

	_, err := m.Reply(fmt.Sprintf("🔇 Playback has been muted by %s.", m.Sender.FirstName), telegram.SendOptions{ReplyMarkup: core.ControlButtons("mute")})
	return err
}

// unmuteHandler handles the /unmute command.
func unmuteHandler(m *telegram.NewMessage) error {
	chatId, _ := getPeerId(m.Client, m.ChatID())
	if !cache.ChatCache.IsActive(chatId) {
		_, err := m.Reply("⏸ There is no track currently playing.")
		return err
	}

	if _, err := vc.Calls.Unmute(chatId); err != nil {
		_, _ = m.Reply("❌ An error occurred while unmuting the playback: " + err.Error())
		return err
	}

	_, err := m.Reply(fmt.Sprintf("🔊 Playback has been unmuted by %s.", m.Sender.FirstName), telegram.SendOptions{ReplyMarkup: core.ControlButtons("unmute")})
	return err
}
