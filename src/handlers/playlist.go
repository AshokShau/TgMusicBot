/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/core/dl"
	"ashokshau/tgmusic/src/lang"

	"github.com/amarnathcjd/gogram/telegram"
)

func createPlaylistHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	userID := m.SenderID()
	ctx, cancel := db.Ctx()
	defer cancel()

	langCode := db.Instance.GetLang(ctx, chatID)
	args := m.Args()
	if args == "" {
		_, err := m.Reply(lang.GetString(langCode, "playlist_create_usage"))
		return err
	}

	userPlaylists, err := db.Instance.GetUserPlaylists(ctx, userID)
	if err != nil {
		_, err := m.Reply(lang.GetString(langCode, "playlist_create_error"))
		return err
	}

	if len(userPlaylists) >= 10 {
		_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_create_limit"), 10))
		return telegram.EndGroup
	}

	if len([]rune(args)) > 40 {
		args = string([]rune(args)[:40])
	}

	playlistID, err := db.Instance.CreatePlaylist(ctx, args, userID)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_create_error"), err.Error()))
		return err
	}

	_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_created"), args, playlistID))
	return telegram.EndGroup
}

func deletePlaylistHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	userID := m.SenderID()
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	args := m.Args()
	if args == "" {
		_, err := m.Reply(lang.GetString(langCode, "playlist_delete_usage"))
		return err
	}
	playlist, err := db.Instance.GetPlaylist(ctx, args)
	if err != nil {
		_, err := m.Reply(lang.GetString(langCode, "playlist_not_found"))
		return err
	}
	if playlist.UserID != userID {
		_, err := m.Reply(lang.GetString(langCode, "playlist_not_owner"))
		return err
	}

	err = db.Instance.DeletePlaylist(ctx, args, userID)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_delete_error"), err.Error()))
		return err
	}

	_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_deleted"), playlist.Name))
	return err
}

func addToPlaylistHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	userID := m.SenderID()
	ctx, cancel := db.Ctx()
	defer cancel()

	langCode := db.Instance.GetLang(ctx, chatID)
	args := strings.SplitN(m.Args(), " ", 2)
	if len(args) != 2 {
		_, err := m.Reply(lang.GetString(langCode, "playlist_add_usage"))
		return err
	}
	playlistID := args[0]
	songURL := args[1]
	playlist, err := db.Instance.GetPlaylist(ctx, playlistID)
	if err != nil {
		_, err := m.Reply(lang.GetString(langCode, "playlist_not_found"))
		return err
	}
	if playlist.UserID != userID {
		_, err := m.Reply(lang.GetString(langCode, "playlist_not_owner"))
		return err
	}
	wrapper := dl.NewDownloaderWrapper(songURL)
	if !wrapper.IsValid() {
		_, err := m.Reply(lang.GetString(langCode, "play_invalid_url"))
		return err
	}
	trackInfo, err := wrapper.GetInfo(ctx)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf(lang.GetString(langCode, "play_fetch_error"), err.Error()))
		return err
	}

	if trackInfo.Results == nil {
		_, err := m.Reply(lang.GetString(langCode, "play_no_tracks_found"))
		return err
	}

	song := db.Song{
		URL:      trackInfo.Results[0].URL,
		Name:     trackInfo.Results[0].Name,
		TrackID:  trackInfo.Results[0].ID,
		Duration: trackInfo.Results[0].Duration,
		Platform: trackInfo.Results[0].Platform,
	}

	err = db.Instance.AddSongToPlaylist(ctx, playlistID, song)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_add_error"), err.Error()))
		return err
	}
	_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_song_added"), song.Name, playlist.Name))
	return err
}

func removeFromPlaylistHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	userID := m.SenderID()
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	args := strings.SplitN(m.Args(), " ", 2)
	if len(args) != 2 {
		_, err := m.Reply(lang.GetString(langCode, "playlist_remove_usage"))
		return err
	}
	playlistID := args[0]
	songIdentifier := args[1]
	playlist, err := db.Instance.GetPlaylist(ctx, playlistID)
	if err != nil {
		_, err := m.Reply(lang.GetString(langCode, "playlist_not_found"))
		return err
	}

	if playlist.UserID != userID {
		_, err := m.Reply(lang.GetString(langCode, "playlist_not_owner"))
		return err
	}

	songIndex, err := strconv.Atoi(songIdentifier)
	var trackID string
	if err == nil {
		if songIndex < 1 || songIndex > len(playlist.Songs) {
			_, err := m.Reply(lang.GetString(langCode, "playlist_remove_invalid_index"))
			return err
		}

		trackID = playlist.Songs[songIndex-1].TrackID
	} else {
		for _, song := range playlist.Songs {
			if song.URL == songIdentifier || song.TrackID == songIdentifier {
				trackID = song.TrackID
				break
			}
		}
	}

	if trackID == "" {
		_, err := m.Reply(lang.GetString(langCode, "playlist_remove_song_not_found"))
		return err
	}

	logger.Info("Removing song from playlist %s: %s", playlistID, trackID)
	err = db.Instance.RemoveSongFromPlaylist(ctx, playlistID, trackID)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_remove_error"), err.Error()))
		return err
	}

	_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_song_removed"), playlist.Name))
	return err
}

func playlistInfoHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	args := m.Args()
	if args == "" {
		_, err := m.Reply(lang.GetString(langCode, "playlist_info_usage"))
		return err
	}

	playlist, err := db.Instance.GetPlaylist(ctx, args)
	if err != nil {
		_, err := m.Reply(lang.GetString(langCode, "playlist_not_found"))
		return err
	}
	var songs []string
	for i, song := range playlist.Songs {
		songs = append(songs, fmt.Sprintf("%d. %s (%s)", i+1, song.Name, song.URL))
	}
	owner, err := m.Client.GetUser(playlist.UserID)
	if err != nil {
		logger.Warn(err.Error())
		return telegram.EndGroup
	}

	_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_info"), playlist.Name, owner.FirstName, len(playlist.Songs), strings.Join(songs, "\n")))
	return telegram.EndGroup
}

func myPlaylistsHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	userID := m.SenderID()
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	playlists, err := db.Instance.GetUserPlaylists(ctx, userID)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_fetch_error"), err.Error()))
		return err
	}
	if len(playlists) == 0 {
		_, err := m.Reply(lang.GetString(langCode, "playlist_no_playlists"))
		return err
	}
	var playlistInfo []string
	for _, playlist := range playlists {
		playlistInfo = append(playlistInfo, fmt.Sprintf("- %s (<code>%s</code>)", playlist.Name, playlist.ID))
	}
	_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "playlist_my_playlists"), strings.Join(playlistInfo, "\n")))
	return err
}
