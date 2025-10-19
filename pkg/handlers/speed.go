package handlers

import (
	"fmt"
	"strconv"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"
	"github.com/AshokShau/TgMusicBot/pkg/vc"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// speedHandler handles the /speed command.
func speedHandler(m *tg.NewMessage) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.Reply(lang.GetString(langCode, "no_track_playing"))
		return err
	}

	if playingSong := cache.ChatCache.GetPlayingTrack(chatID); playingSong == nil {
		_, err := m.Reply(lang.GetString(langCode, "no_track_playing"))
		return err
	}

	args := m.Args()
	if args == "" {
		_, _ = m.Reply(lang.GetString(langCode, "speed_usage"))
		return nil
	}

	speed, err := strconv.ParseFloat(args, 64)
	if err != nil {
		_, _ = m.Reply(lang.GetString(langCode, "speed_invalid_value"))
		return nil
	}

	if speed < 0.5 || speed > 4.0 {
		_, _ = m.Reply(lang.GetString(langCode, "speed_out_of_range"))
		return nil
	}

	if err = vc.Calls.ChangeSpeed(chatID, speed); err != nil {
		_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "speed_error"), err.Error()))
		return nil
	}
	_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "speed_success"), speed))
	return nil
}
