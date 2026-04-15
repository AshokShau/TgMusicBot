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

	if !cache.ChatCache.IsActive(chatID) {
		text := "There is no active playback."
		_ = cb.Answer(c, 0, false, text, "")
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatID)
	if currentTrack == nil {
		_ = cb.Answer(c, 0, false, "There is no active playback.", "")
		_, _ = cb.EditMessageText(c, "There is no active playback.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	buildTrackMessage := func(status, emoji string) string {
		return fmt.Sprintf("%s <b>%s</b>\n\n<b>Track:</b> <a href='%s'>%s</a>\n<b>Duration:</b> %s\n<b>Requested by:</b> %s",
			emoji, status,
			currentTrack.URL, currentTrack.Name,
			utils.SecToMin(currentTrack.Duration),
			currentTrack.User,
		)
	}

	switch {
	case strings.Contains(data, "play_skip"):
		if err := vc.Calls.PlayNext(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to skip the current track.", "")
			_, _ = cb.EditMessageText(c, "Unable to skip the current track.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, 0, false, "Track skipped.", "")
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil

	case strings.Contains(data, "play_stop"):
		if err := vc.Calls.Stop(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to stop playback.", "")
			_, _ = cb.EditMessageText(c, "Unable to stop playback.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		msg := fmt.Sprintf("<b>Playback stopped.</b>\nRequested by: %s", user.FirstName)
		_ = cb.Answer(c, 0, false, "Playback stopped.", "")
		_, err := cb.EditMessageText(c, msg, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
		return err

	case strings.Contains(data, "play_pause"):
		if _, err = vc.Calls.Pause(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to pause playback.", "")
			_, _ = cb.EditMessageText(c, "Unable to pause playback.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, 0, false, "Playback paused.", "")
		text := buildTrackMessage("Paused", "⏸") + fmt.Sprintf("\n\nPaused by %s", user.FirstName)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("pause"), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_resume"):
		if _, err := vc.Calls.Resume(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to resume playback.", "")
			_, _ = cb.EditMessageText(c, "Unable to resume playback.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("pause"), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, 0, false, "Playback resumed.", "")
		text := buildTrackMessage("Now Playing", "▶") + fmt.Sprintf("\n\nResumed by %s", user.FirstName)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("resume"), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_mute"):
		if _, err := vc.Calls.Mute(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to mute playback.", "")
			_, _ = cb.EditMessageText(c, "Unable to mute playback.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("mute"), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, 0, false, "Playback muted.", "")
		text := buildTrackMessage("Muted", "🔇") + fmt.Sprintf("\n\nMuted by %s", user.FirstName)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("mute"), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_unmute"):
		if _, err := vc.Calls.Unmute(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to unmute playback.", "")
			_, _ = cb.EditMessageText(c, "Unable to unmute playback.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("unmute"), ParseMode: "HTML"})
			return nil
		}
		_ = cb.Answer(c, 0, false, "Playback unmuted.", "")
		text := buildTrackMessage("Now Playing", "▶") + fmt.Sprintf("\n\nUnmuted by %s", user.FirstName)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("unmute"), DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_add_to_list"):
		playlists, err := db.Instance.GetUserPlaylists(cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, 0, false, "Unable to fetch playlists.", "")
			return nil
		}

		var playlistID string
		if len(playlists) == 0 {
			playlistID, err = db.Instance.CreatePlaylist("My Playlist (TgMusic)", cb.SenderUserId)
			if err != nil {
				_ = cb.Answer(c, 0, false, "Unable to create playlist.", "")
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

		err = db.Instance.AddSongToPlaylist(playlistID, song)
		if err != nil {
			_ = cb.Answer(c, 0, false, "Unable to add track to playlist.", "")
			return nil
		}

		playlist, err := db.Instance.GetPlaylist(playlistID)
		if err != nil {
			_ = cb.Answer(c, 0, false, "Playlist not found.", "")
			return nil
		}

		_ = cb.Answer(c, 0, false, fmt.Sprintf("Track \"%s\" added to playlist \"%s\".", song.Name, playlist.Name), "")
		return nil
	}

	text := buildTrackMessage("Now Playing", "▶")
	_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("resume"), ParseMode: "HTML", DisableWebPagePreview: true})
	return nil
}

func vcPlayHandler(c *td.Client, ctx *td.Context) error {
	cb := ctx.Update.UpdateNewCallbackQuery
	data := cb.DataString()

	if strings.Contains(data, "vcplay_close") {
		_ = cb.Answer(c, 0, false, "Closing panel.", "")
		_ = c.DeleteMessages(cb.ChatId, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	slog.Info("Received vcplay callback", "arg1", data)
	return nil
}
