/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tgmusic/internal/cache"
	"ashokshau/tgmusic/internal/calls"
	"ashokshau/tgmusic/internal/db"
	"ashokshau/tgmusic/internal/utils"
	"fmt"
	"html"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func playCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()
	if !adminModeCB(c, cb) {
		return td.EndGroups
	}

	chatID := cb.ChatId
	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "Unknown", Id: cb.SenderUserId}
	}

	if !cache.ChatCache.IsActive(chatID) {
		text := "There is no active playback."
		_ = cb.Answer(c, text, nil)
		_, _ = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatID)
	if currentTrack == nil {
		_ = cb.Answer(c, "There is no active playback.", nil)
		_, _ = cb.EditMessageText(c, "There is no active playback.", &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	buildTrackMessage := func(status, emoji string) string {
		escURL := html.EscapeString(currentTrack.URL)
		escName := html.EscapeString(currentTrack.Name)
		escUser := html.EscapeString(currentTrack.User)
		return fmt.Sprintf("%s <b>%s</b>\n\n<b>Track:</b> <a href='%s'>%s</a>\n<b>Duration:</b> %s\n<b>Requested by:</b> %s",
			emoji, status,
			escURL, escName,
			utils.SecToMin(currentTrack.Duration),
			escUser,
		)
	}

	switch {
	case strings.Contains(data, "play_skip"):
		if err := calls.Calls.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, "Unable to skip the current track.", nil)
			_, _ = cb.EditMessageText(c, "Unable to skip the current track.", &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, "Track skipped.", nil)
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil

	case strings.Contains(data, "play_stop"):
		if err := calls.Calls.Stop(chatID, false); err != nil {
			_ = cb.Answer(c, "Unable to stop playback.", nil)
			_, _ = cb.EditMessageText(c, "Unable to stop playback.", &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}

		msg := fmt.Sprintf("<b>Playback stopped.</b>\nRequested by: %s", html.EscapeString(user.FirstName))
		_ = cb.Answer(c, "Playback stopped.", nil)
		_, err := cb.EditMessageText(c, msg, &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
		return err

	case strings.Contains(data, "play_pause"):
		if _, err = calls.Calls.Pause(chatID); err != nil {
			_ = cb.Answer(c, "Unable to pause playback.", nil)
			_, _ = cb.EditMessageText(c, "Unable to pause playback.", &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, "Playback paused.", nil)
		text := buildTrackMessage("Paused", "⏸") + fmt.Sprintf("\n\nPaused by %s", html.EscapeString(user.FirstName))
		_, _ = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons("pause"), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_resume"):
		if _, err := calls.Calls.Resume(chatID); err != nil {
			_ = cb.Answer(c, "Unable to resume playback.", nil)
			_, _ = cb.EditMessageText(c, "Unable to resume playback.", &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons("pause"), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, "Playback resumed.", nil)
		text := buildTrackMessage("Now Playing", "▶") + fmt.Sprintf("\n\nResumed by %s", html.EscapeString(user.FirstName))
		_, _ = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons("resume"), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_mute"):
		if _, err := calls.Calls.Mute(chatID); err != nil {
			_ = cb.Answer(c, "Unable to mute playback.", nil)
			_, _ = cb.EditMessageText(c, "Unable to mute playback.", &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons("mute"), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, "Playback muted.", nil)
		text := buildTrackMessage("Muted", "🔇") + fmt.Sprintf("\n\nMuted by %s", html.EscapeString(user.FirstName))
		_, _ = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons("mute"), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_unmute"):
		if _, err := calls.Calls.Unmute(chatID); err != nil {
			_ = cb.Answer(c, "Unable to unmute playback.", nil)
			_, _ = cb.EditMessageText(c, "Unable to unmute playback.", &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons("unmute"), ParseMode: "HTML"})
			return nil
		}
		_ = cb.Answer(c, "Playback unmuted.", nil)
		text := buildTrackMessage("Now Playing", "▶") + fmt.Sprintf("\n\nUnmuted by %s", html.EscapeString(user.FirstName))
		_, _ = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons("unmute"), DisableWebPagePreview: true})
		return nil

	case strings.Contains(data, "play_add_to_list"):
		playlists, err := db.Instance.GetUserPlaylists(cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, "Unable to fetch playlists.", nil)
			return nil
		}

		var playlistID string
		if len(playlists) == 0 {
			playlistID, err = db.Instance.CreatePlaylist("My Playlist (TgMusic)", cb.SenderUserId)
			if err != nil {
				_ = cb.Answer(c, "Unable to create playlist.", nil)
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
			_ = cb.Answer(c, "Unable to add track to playlist.", nil)
			return nil
		}

		playlist, err := db.Instance.GetPlaylist(playlistID)
		if err != nil {
			_ = cb.Answer(c, "Playlist not found.", nil)
			return nil
		}

		_ = cb.Answer(c, fmt.Sprintf("Track \"%s\" added to playlist \"%s\".", song.Name, playlist.Name), nil)
		return nil

	case strings.HasPrefix(data, "play_now_"):
		trackID := strings.TrimPrefix(data, "play_now_")
		if strings.HasPrefix(data, "play_now_hash_") {
			trackID = ""
			for _, track := range cache.ChatCache.GetQueue(chatID) {
				if track != nil && utils.PlayNowCallbackData(track.TrackID) == data {
					trackID = track.TrackID
					break
				}
			}
		}
		if ok := cache.ChatCache.MoveTrackToFront(chatID, trackID); !ok {
			_ = cb.Answer(c, "Track not found in queue.", nil)
			return nil
		}

		_ = cb.Answer(c, "Playing now.", nil)
		if err = calls.Calls.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, "Unable to play the track.", nil)
			return nil
		}

		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	text := buildTrackMessage("Now Playing", "▶")
	_, _ = cb.EditMessageText(c, text, &td.EditMessageTextOpts{ReplyMarkup: utils.ControlButtons("resume"), ParseMode: "HTML", DisableWebPagePreview: true})
	return nil
}

func vcPlayHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()

	if strings.Contains(data, "vcplay_close") {
		_ = cb.Answer(c, "Closing panel.", nil)
		_ = c.DeleteMessages(cb.ChatId, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	c.Logger.Info("Received vcplay callback", "data", data)
	return nil
}
