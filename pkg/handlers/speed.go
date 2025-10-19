package handlers

import (
	"fmt"
	"strconv"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/vc"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// speedHandler handles the /speed command.
func speedHandler(m *tg.NewMessage) error {
	chatId, _ := getPeerId(m.Client, m.ChatID())
	if !cache.ChatCache.IsActive(chatId) {
		_, err := m.Reply("⏸ There is no track currently playing.")
		return err
	}

	if playingSong := cache.ChatCache.GetPlayingTrack(chatId); playingSong == nil {
		_, err := m.Reply("⏸ There is no track currently playing.")
		return err
	}

	args := m.Args()
	if args == "" {
		_, _ = m.Reply("<b>❌ Change Speed</b>\n\n<b>Usage:</b> <code>/speed [value]</code>\n\n- The speed can be set from <code>0.5</code> to <code>4.0</code>.")
		return nil
	}

	speed, err := strconv.ParseFloat(args, 64)
	if err != nil {
		_, _ = m.Reply("❌ Invalid speed value provided. Please use a number between 0.5 and 4.0.")
		return nil
	}

	if speed < 0.5 || speed > 4.0 {
		_, _ = m.Reply("⚠️ The speed must be between 0.5 and 4.0.")
		return nil
	}

	if err = vc.Calls.ChangeSpeed(chatId, speed); err != nil {
		_, _ = m.Reply("❌ An error occurred while changing the speed: " + err.Error())
		return nil
	}
	_, _ = m.Reply(fmt.Sprintf("✅ The playback speed has been changed to %.2fx.", speed))
	return nil
}
