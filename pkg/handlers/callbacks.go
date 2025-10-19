package handlers

import (
	"fmt"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"
	"github.com/AshokShau/TgMusicBot/pkg/vc"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

// playCallbackHandler handles callbacks from the play keyboard.
// It takes a telegram.CallbackQuery object as input.
// It returns an error if any.
func playCallbackHandler(cb *telegram.CallbackQuery) error {
	data := cb.DataString()
	if strings.Contains(data, "settings_") {
		return nil
	}

	chatID, _ := getPeerId(cb.Client, cb.ChatID)
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if !cache.ChatCache.IsActive(chatID) {
		text := lang.GetString(langCode, "no_track_playing")
		_, _ = cb.Answer(text, &telegram.CallbackOptions{Alert: true})
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatID)
	if currentTrack == nil {
		_, _ = cb.Answer(lang.GetString(langCode, "no_track_playing"), &telegram.CallbackOptions{Alert: true})
		_, _ = cb.Edit(lang.GetString(langCode, "no_track_playing"), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
		return nil
	}

	buildTrackMessage := func(status, emoji string) string {
		return fmt.Sprintf(
			lang.GetString(langCode, "track_message"),
			emoji, status,
			currentTrack.URL, currentTrack.Name,
			cache.SecToMin(currentTrack.Duration),
			currentTrack.User,
		)
	}

	switch {
	case strings.Contains(data, "play_skip"):
		if err := vc.Calls.PlayNext(chatID); err != nil {
			_, _ = cb.Answer(lang.GetString(langCode, "skip_fail"), &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit(lang.GetString(langCode, "skip_fail"), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
			return nil
		}
		_, _ = cb.Answer(lang.GetString(langCode, "track_skipped"), &telegram.CallbackOptions{Alert: true})
		_, _ = cb.Delete()
		return nil

	case strings.Contains(data, "play_stop"):
		if err := vc.Calls.Stop(chatID); err != nil {
			_, _ = cb.Answer(lang.GetString(langCode, "stop_fail"), &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit(lang.GetString(langCode, "stop_fail"), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
			return nil
		}
		msg := fmt.Sprintf(lang.GetString(langCode, "playback_stopped"), cb.Sender.FirstName)
		_, _ = cb.Answer(lang.GetString(langCode, "track_stopped"), &telegram.CallbackOptions{Alert: true})
		_, err := cb.Edit(msg, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
		return err

	case strings.Contains(data, "play_pause"):
		if _, err := vc.Calls.Pause(chatID); err != nil {
			_, _ = cb.Answer(lang.GetString(langCode, "pause_fail"), &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit(lang.GetString(langCode, "pause_fail"), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
			return nil
		}
		_, _ = cb.Answer(lang.GetString(langCode, "track_paused"), &telegram.CallbackOptions{Alert: true})
		text := buildTrackMessage(lang.GetString(langCode, "paused"), "⏸") + fmt.Sprintf(lang.GetString(langCode, "paused_by"), cb.Sender.FirstName)
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("pause")})
		return nil

	case strings.Contains(data, "play_resume"):
		if _, err := vc.Calls.Resume(chatID); err != nil {
			_, _ = cb.Answer(lang.GetString(langCode, "resume_fail"), &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit(lang.GetString(langCode, "resume_fail"), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("pause")})
			return nil
		}
		_, _ = cb.Answer(lang.GetString(langCode, "track_resumed"), &telegram.CallbackOptions{Alert: true})
		text := buildTrackMessage(lang.GetString(langCode, "now_playing"), "🎵") + fmt.Sprintf(lang.GetString(langCode, "resumed_by"), cb.Sender.FirstName)
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("resume")})
		return nil

	case strings.Contains(data, "play_mute"):
		if _, err := vc.Calls.Mute(chatID); err != nil {
			_, _ = cb.Answer(lang.GetString(langCode, "mute_fail"), &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit(lang.GetString(langCode, "mute_fail"), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("mute")})
			return nil
		}
		_, _ = cb.Answer(lang.GetString(langCode, "track_muted"), &telegram.CallbackOptions{Alert: true})
		text := buildTrackMessage(lang.GetString(langCode, "muted"), "🔇") + fmt.Sprintf(lang.GetString(langCode, "muted_by"), cb.Sender.FirstName)
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("mute")})
		return nil

	case strings.Contains(data, "play_unmute"):
		if _, err := vc.Calls.Unmute(chatID); err != nil {
			_, _ = cb.Answer(lang.GetString(langCode, "unmute_fail"), &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit(lang.GetString(langCode, "unmute_fail"), &telegram.SendOptions{ReplyMarkup: core.ControlButtons("unmute")})
			return nil
		}
		_, _ = cb.Answer(lang.GetString(langCode, "track_unmuted"), &telegram.CallbackOptions{Alert: true})
		text := buildTrackMessage(lang.GetString(langCode, "now_playing"), "🎵") + fmt.Sprintf(lang.GetString(langCode, "unmuted_by"), cb.Sender.FirstName)
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("unmute")})
		return nil
	}

	text := buildTrackMessage(lang.GetString(langCode, "now_playing"), "🎵")
	_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("resume")})
	return nil
}

// vcPlayHandler handles callbacks from the vcplay keyboard.
// It takes a telegram.CallbackQuery object as input.
// It returns an error if any.
func vcPlayHandler(cb *telegram.CallbackQuery) error {
	chatID, _ := getPeerId(cb.Client, cb.ChatID)
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	data := cb.DataString()
	if strings.Contains(data, "vcplay_close") {
		_, _ = cb.Answer(lang.GetString(langCode, "closed"), &telegram.CallbackOptions{Alert: true})
		_, _ = cb.Delete()
		return nil
	}
	gologging.InfoF("vcPlayHandler: %s", data)
	return nil
}
