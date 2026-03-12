/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/utils"
	"fmt"
	"log/slog"
	"strings"

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// playCallbackHandler handles callbacks from the play keyboard.
// It returns an error if any.
func playCallbackHandler(c *td.Client, ctx *td.Context) error {
	cb := ctx.Update.UpdateNewCallbackQuery
	if !adminModeCB(c, cb) {
		return td.EndGroups
	}

	data := cb.DataString()
	if strings.Contains(data, "settings_") {
		return nil
	}

	chatID := cb.ChatId
	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "Unknown", Id: cb.SenderUserId}
	}

	ctx2, cancel := db.Ctx()
	defer cancel()

	if !cache.ChatCache.IsActive(chatID) {
		text := "⏸ No track currently playing."
		_ = cb.Answer(c, 300, false, text, "")
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML"})
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatID)
	if currentTrack == nil {
		_ = cb.Answer(c, 300, false, "⏸ No track currently playing.", "")
		_, _ = cb.EditMessageText(c, "⏸ No track currently playing.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML"})
		return nil
	}

	buildTrackMessage := func(status, emoji string) string {
		return fmt.Sprintf(
			"%s <b>%s</b>\n\n🎧 <b>Track:</b> <a href='%s'>%s</a>\n🕒 <b>Duration:</b> %s\n🙋‍♂️ <b>Requested by:</b> %s",
			emoji, status,
			currentTrack.URL, currentTrack.Name,
			utils.SecToMin(currentTrack.Duration),
			currentTrack.User,
		)
	}

	switch {
	case strings.Contains(data, "play_skip"):
		if err := vc.Calls.PlayNext(chatID); err != nil {
			_ = cb.Answer(c, 300, false, "Failed to skip track.", "")
			_, _ = cb.EditMessageText(c, "Failed to skip track.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML"})
			return nil
		}
		_ = cb.Answer(c, 300, false, "Track skipped.", "")
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil

	case strings.Contains(data, "play_stop"):
		if err := vc.Calls.Stop(chatID); err != nil {
			_ = cb.Answer(c, 300, false, "Failed to stop track.", "")
			_, _ = cb.EditMessageText(c, "Failed to stop track.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML"})
			return nil
		}
		msg := fmt.Sprintf("⏹ <b>Playback Stopped</b>\n└ Requested by: %s", user.FirstName)
		_ = cb.Answer(c, 300, false, "Playback stopped.", "")
		_, err := cb.EditMessageText(c, msg, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML"})
		return err

	case strings.Contains(data, "play_pause"):
		if _, err = vc.Calls.Pause(chatID); err != nil {
			_ = cb.Answer(c, 300, false, "Failed to pause track.", "")
			_, _ = cb.EditMessageText(c, "Failed to pause track.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML"})
			return nil
		}
		_ = cb.Answer(c, 300, false, "Track paused.", "")
		text := buildTrackMessage("Paused", "⏸") + fmt.Sprintf("\n\n⏸ <i>Paused by %s</i>", user.FirstName)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("pause"), ParseMode: "HTML"})
		return nil

	case strings.Contains(data, "play_resume"):
		if _, err := vc.Calls.Resume(chatID); err != nil {
			_ = cb.Answer(c, 300, false, "Failed to resume track.", "")
			_, _ = cb.EditMessageText(c, "Failed to resume track.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("pause"), ParseMode: "HTML"})
			return nil
		}

		_ = cb.Answer(c, 300, false, "Track resumed.", "")
		text := buildTrackMessage("Now Playing", "🎵") + fmt.Sprintf("\n\n▶️ <i>Resumed by %s</i>", user.FirstName)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("resume"), ParseMode: "HTML"})
		return nil

	case strings.Contains(data, "play_mute"):
		if _, err := vc.Calls.Mute(chatID); err != nil {
			_ = cb.Answer(c, 300, false, "Failed to mute track.", "")
			_, _ = cb.EditMessageText(c, "Failed to mute track.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("mute"), ParseMode: "HTML"})
			return nil
		}

		_ = cb.Answer(c, 300, false, "Track muted.", "")
		text := buildTrackMessage("Muted", "🔇") + fmt.Sprintf("\n\n🔇 <i>Muted by %s</i>", user.FirstName)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("mute"), ParseMode: "HTML"})
		return nil

	case strings.Contains(data, "play_unmute"):
		if _, err := vc.Calls.Unmute(chatID); err != nil {
			_ = cb.Answer(c, 300, false, "Failed to unmute track.", "")
			_, _ = cb.EditMessageText(c, "Failed to unmute track.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("unmute"), ParseMode: "HTML"})
			return nil
		}
		_ = cb.Answer(c, 300, false, "Track unmuted.", "")
		text := buildTrackMessage("Now Playing", "🎵") + fmt.Sprintf("\n\n🔊 <i>Unmuted by %s</i>", user.FirstName)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("unmute"), DisableWebPagePreview: true})
		return nil
	case strings.Contains(data, "play_add_to_list"):
		playlists, err := db.Instance.GetUserPlaylists(ctx2, cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, 300, false, "An error occurred while fetching playlists.", "")
			return nil
		}

		var playlistID string
		if len(playlists) == 0 {
			playlistID, err = db.Instance.CreatePlaylist(ctx2, "My Playlist (TgMusic)", cb.SenderUserId)
			if err != nil {
				_ = cb.Answer(c, 300, false, "An error occurred while creating a new playlist.", "")
				return nil
			}
		} else {
			playlistID = playlists[0].ID
		}

		song := db.Song{
			URL:      currentTrack.URL,
			Name:     currentTrack.Name,
			TrackID:  currentTrack.TrackID,
			Duration: currentTrack.Duration,
			Platform: currentTrack.Platform,
		}

		err = db.Instance.AddSongToPlaylist(ctx2, playlistID, song)
		if err != nil {
			_ = cb.Answer(c, 300, false, "An error occurred while adding the track to the playlist.", "")
			return nil
		}

		playlist, err := db.Instance.GetPlaylist(ctx2, playlistID)
		if err != nil {
			_ = cb.Answer(c, 300, false, "Playlist not found :)", "")
			return nil
		}

		_ = cb.Answer(c, 300, false, fmt.Sprintf("✅ '%s' has been added to the playlist '%s'.", song.Name, playlist.Name), "")
		return nil
	}

	text := buildTrackMessage("Now Playing", "🎵")
	_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("resume"), ParseMode: "HTML", DisableWebPagePreview: true})
	return nil
}

// vcPlayHandler handles callbacks from the vcplay keyboard.
// It returns an error if any.
func vcPlayHandler(c *td.Client, ctx *td.Context) error {
	cb := ctx.Update.UpdateNewCallbackQuery
	data := cb.DataString()
	if strings.Contains(data, "vcplay_close") {
		_ = cb.Answer(c, 300, false, "Closing...", "")
		_ = c.DeleteMessages(cb.ChatId, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	slog.Info("Received vcplay callback", "arg1", data)
	return nil
}
