package handlers

import (
	"fmt"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
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

	chatId, _ := getPeerId(cb.Client, cb.ChatID)
	if !cache.ChatCache.IsActive(chatId) {
		text := "⏸ No track currently playing."
		_, _ = cb.Answer(text, &telegram.CallbackOptions{Alert: true})
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatId)
	if currentTrack == nil {
		_, _ = cb.Answer("No track currently playing.", &telegram.CallbackOptions{Alert: true})
		_, _ = cb.Edit("No track currently playing.", &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
		return nil
	}

	buildTrackMessage := func(status, emoji string) string {
		return fmt.Sprintf(
			"%s <b>%s</b>\n\n"+
				"🎧 <b>Track:</b> <a href='%s'>%s</a>\n"+
				"🕒 <b>Duration:</b> %s\n"+
				"🙋‍♂️ <b>Requested by:</b> %s",
			emoji, status,
			currentTrack.URL, currentTrack.Name,
			cache.SecToMin(currentTrack.Duration),
			currentTrack.User,
		)
	}

	switch {
	case strings.Contains(data, "play_skip"):
		if err := vc.Calls.PlayNext(chatId); err != nil {
			_, _ = cb.Answer("Failed to skip track.", &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit("Failed to skip track.", &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
			return nil
		}
		_, _ = cb.Answer("Track skipped.", &telegram.CallbackOptions{Alert: true})
		_, _ = cb.Delete()
		return nil

	case strings.Contains(data, "play_stop"):
		if err := vc.Calls.Stop(chatId); err != nil {
			_, _ = cb.Answer("Failed to stop track.", &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit("Failed to stop track.", &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
			return nil
		}
		msg := fmt.Sprintf("⏹ <b>Playback Stopped</b>\n└ Requested by: %s", cb.Sender.FirstName)
		_, _ = cb.Answer("Track stopped.", &telegram.CallbackOptions{Alert: true})
		_, err := cb.Edit(msg, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
		return err

	case strings.Contains(data, "play_pause"):
		if _, err := vc.Calls.Pause(chatId); err != nil {
			_, _ = cb.Answer("Failed to pause track.", &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit("Failed to pause track.", &telegram.SendOptions{ReplyMarkup: core.ControlButtons("")})
			return nil
		}
		_, _ = cb.Answer("Track paused.", &telegram.CallbackOptions{Alert: true})
		text := buildTrackMessage("Paused", "⏸") + fmt.Sprintf("\n\n⏸ <i>Paused by %s</i>", cb.Sender.FirstName)
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("pause")})
		return nil

	case strings.Contains(data, "play_resume"):
		if _, err := vc.Calls.Resume(chatId); err != nil {
			_, _ = cb.Answer("Failed to resume track.", &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit("Failed to resume track.", &telegram.SendOptions{ReplyMarkup: core.ControlButtons("pause")})
			return nil
		}
		_, _ = cb.Answer("Track resumed.", &telegram.CallbackOptions{Alert: true})
		text := buildTrackMessage("Now Playing", "🎵") + fmt.Sprintf("\n\n▶️ <i>Resumed by %s</i>", cb.Sender.FirstName)
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("resume")})
		return nil

	case strings.Contains(data, "play_mute"):
		if _, err := vc.Calls.Mute(chatId); err != nil {
			_, _ = cb.Answer("Failed to mute track.", &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit("Failed to mute track.", &telegram.SendOptions{ReplyMarkup: core.ControlButtons("mute")})
			return nil
		}
		_, _ = cb.Answer("Track muted.", &telegram.CallbackOptions{Alert: true})
		text := buildTrackMessage("Muted", "🔇") + fmt.Sprintf("\n\n🔇 <i>Muted by %s</i>", cb.Sender.FirstName)
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("mute")})
		return nil

	case strings.Contains(data, "play_unmute"):
		if _, err := vc.Calls.Unmute(chatId); err != nil {
			_, _ = cb.Answer("Failed to unmute track.", &telegram.CallbackOptions{Alert: true})
			_, _ = cb.Edit("Failed to unmute track.", &telegram.SendOptions{ReplyMarkup: core.ControlButtons("unmute")})
			return nil
		}
		_, _ = cb.Answer("Track unmuted.", &telegram.CallbackOptions{Alert: true})
		text := buildTrackMessage("Now Playing", "🎵") + fmt.Sprintf("\n\n🔊 <i>Unmuted by %s</i>", cb.Sender.FirstName)
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("unmute")})
		return nil
	}

	text := buildTrackMessage("Now Playing", "🎵")
	_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("resume")})
	return nil
}

// vcPlayHandler handles callbacks from the vcplay keyboard.
// It takes a telegram.CallbackQuery object as input.
// It returns an error if any.
func vcPlayHandler(cb *telegram.CallbackQuery) error {
	// chatId, _ := getPeerId(cb.Client, cb.ChatID)
	data := cb.DataString()
	if strings.Contains(data, "vcplay_close") {
		_, _ = cb.Answer("Closed !", &telegram.CallbackOptions{Alert: true})
		_, _ = cb.Delete()
		return nil
	}
	gologging.InfoF("vcPlayHandler: %s", data)
	return nil
}
