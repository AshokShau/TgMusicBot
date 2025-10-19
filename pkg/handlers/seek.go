package handlers

import (
	"fmt"
	"strconv"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"
	"github.com/AshokShau/TgMusicBot/pkg/vc"

	"github.com/amarnathcjd/gogram/telegram"
)

// seekHandler handles the /seek command.
func seekHandler(m *telegram.NewMessage) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.Reply(lang.GetString(langCode, "no_track_playing"))
		return err
	}

	playingSong := cache.ChatCache.GetPlayingTrack(chatID)
	if playingSong == nil {
		_, err := m.Reply(lang.GetString(langCode, "no_track_playing"))
		return err
	}

	args := m.Args()
	if args == "" {
		_, _ = m.Reply(lang.GetString(langCode, "seek_usage"))
		return nil
	}

	seekTime, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.Reply(lang.GetString(langCode, "seek_invalid_time"))
		return nil
	}

	if seekTime < 0 || seekTime < 20 {
		_, _ = m.Reply(lang.GetString(langCode, "seek_min_time"))
		return nil
	}

	currDur, err := vc.Calls.PlayedTime(chatID)
	if err != nil {
		_, _ = m.Reply(lang.GetString(langCode, "seek_fetch_duration_error"))
		return nil
	}

	toSeek := int(currDur) + seekTime
	if toSeek >= playingSong.Duration {
		_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "seek_beyond_duration"), cache.SecToMin(playingSong.Duration)))
		return nil
	}

	if err = vc.Calls.SeekStream(chatID, playingSong.FilePath, toSeek, playingSong.Duration, playingSong.IsVideo); err != nil {
		_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "seek_error"), err.Error()))
		return nil
	}

	_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "seek_success"), cache.SecToMin(toSeek)))
	return nil
}
