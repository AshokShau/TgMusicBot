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

	"github.com/amarnathcjd/gogram/telegram"
)

func createPlaylistHandler(m *telegram.NewMessage) error {
	userID := m.SenderID()
	ctx, cancel := db.Ctx()
	defer cancel()

	args := m.Args()
	if args == "" {
		_, err := m.Reply("<b>Usage:</b> /createplaylist [playlist name]")
		return err
	}

	userPlaylists, err := db.Instance.GetUserPlaylists(ctx, userID)
	if err != nil {
		_, err := m.Reply("Error creating playlist.")
		return err
	}

	if len(userPlaylists) >= 10 {
		_, _ = m.Reply(fmt.Sprintf("You have reached the limit of %d playlists.", 10))
		return telegram.ErrEndGroup
	}

	if len([]rune(args)) > 40 {
		args = string([]rune(args)[:40])
	}

	playlistID, err := db.Instance.CreatePlaylist(ctx, args, userID)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf("Error creating playlist: %s", err.Error()))
		return err
	}

	_, err = m.Reply(fmt.Sprintf("✅ Playlist '%s' created (ID: <code>%s</code>).", args, playlistID))
	return telegram.ErrEndGroup
}

func deletePlaylistHandler(m *telegram.NewMessage) error {
	userID := m.SenderID()
	ctx, cancel := db.Ctx()
	defer cancel()
	args := m.Args()
	if args == "" {
		_, err := m.Reply("<b>Usage:</b> /deleteplaylist [playlist id]")
		return err
	}
	playlist, err := db.Instance.GetPlaylist(ctx, args)
	if err != nil {
		_, err := m.Reply("❌ Playlist not found.")
		return err
	}
	if playlist.UserID != userID {
		_, err := m.Reply("❌ You don't own this playlist.")
		return err
	}

	err = db.Instance.DeletePlaylist(ctx, args, userID)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf("Error deleting playlist: %s", err.Error()))
		return err
	}

	_, err = m.Reply(fmt.Sprintf("✅ Playlist '%s' deleted.", playlist.Name))
	return err
}

func addToPlaylistHandler(m *telegram.NewMessage) error {
	userID := m.SenderID()
	ctx, cancel := db.Ctx()
	defer cancel()

	args := strings.SplitN(m.Args(), " ", 2)
	if len(args) != 2 {
		_, err := m.Reply("<b>Usage:</b> /addtoplaylist [playlist id] [song url]")
		return err
	}
	playlistID := args[0]
	songURL := args[1]
	playlist, err := db.Instance.GetPlaylist(ctx, playlistID)
	if err != nil {
		_, err := m.Reply("❌ Playlist not found.")
		return err
	}
	if playlist.UserID != userID {
		_, err := m.Reply("❌ You don't own this playlist.")
		return err
	}
	wrapper := dl.NewDownloaderWrapper(songURL)
	if !wrapper.IsValid() {
		_, err := m.Reply("❌ Invalid URL or unsupported platform.")
		return err
	}
	trackInfo, err := wrapper.GetInfo(ctx)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf("❌ Error fetching track info: %s", err.Error()))
		return err
	}

	if trackInfo.Results == nil {
		_, err := m.Reply("❌ No tracks found.")
		return err
	}

	song := db.Song{
		URL:      trackInfo.Results[0].Url,
		Name:     trackInfo.Results[0].Title,
		TrackID:  trackInfo.Results[0].Id,
		Duration: trackInfo.Results[0].Duration,
		Platform: trackInfo.Results[0].Platform,
	}

	err = db.Instance.AddSongToPlaylist(ctx, playlistID, song)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf("Error adding song: %s", err.Error()))
		return err
	}
	_, err = m.Reply(fmt.Sprintf("✅ '%s' added to playlist '%s'.", song.Name, playlist.Name))
	return err
}

func removeFromPlaylistHandler(m *telegram.NewMessage) error {
	userID := m.SenderID()
	ctx, cancel := db.Ctx()
	defer cancel()
	args := strings.SplitN(m.Args(), " ", 2)
	if len(args) != 2 {
		_, err := m.Reply("<b>Usage:</b> /removefromplaylist [playlist id] [song number or url]")
		return err
	}
	playlistID := args[0]
	songIdentifier := args[1]
	playlist, err := db.Instance.GetPlaylist(ctx, playlistID)
	if err != nil {
		_, err := m.Reply("❌ Playlist not found.")
		return err
	}

	if playlist.UserID != userID {
		_, err := m.Reply("❌ You don't own this playlist.")
		return err
	}

	songIndex, err := strconv.Atoi(songIdentifier)
	var trackID string
	if err == nil {
		if songIndex < 1 || songIndex > len(playlist.Songs) {
			_, err := m.Reply("❌ Invalid song number.")
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
		_, err := m.Reply("❌ Song not found in playlist.")
		return err
	}

	logger.Info("Removing song from playlist %s: %s", playlistID, trackID)
	err = db.Instance.RemoveSongFromPlaylist(ctx, playlistID, trackID)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf("Error removing song: %s", err.Error()))
		return err
	}

	_, err = m.Reply(fmt.Sprintf("✅ Song removed from playlist '%s'.", playlist.Name))
	return err
}

func playlistInfoHandler(m *telegram.NewMessage) error {
	ctx, cancel := db.Ctx()
	defer cancel()
	args := m.Args()
	if args == "" {
		_, err := m.Reply("<b>Usage:</b> /playlistinfo [playlist id]")
		return err
	}

	playlist, err := db.Instance.GetPlaylist(ctx, args)
	if err != nil {
		_, err := m.Reply("❌ Playlist not found.")
		return err
	}
	var songs []string
	for i, song := range playlist.Songs {
		songs = append(songs, fmt.Sprintf("%d. %s (%s)", i+1, song.Name, song.URL))
	}
	owner, err := m.Client.GetUser(playlist.UserID)
	if err != nil {
		logger.Warn(err.Error())
		return telegram.ErrEndGroup
	}

	_, err = m.Reply(fmt.Sprintf("<b>Playlist Info</b>\n\n<b>Name:</b> %s\n<b>Owner:</b> %s\n<b>Songs:</b> %d\n\n%s", playlist.Name, owner.FirstName, len(playlist.Songs), strings.Join(songs, "\n")))
	return telegram.ErrEndGroup
}

func myPlaylistsHandler(m *telegram.NewMessage) error {
	userID := m.SenderID()
	ctx, cancel := db.Ctx()
	defer cancel()
	playlists, err := db.Instance.GetUserPlaylists(ctx, userID)
	if err != nil {
		_, err := m.Reply(fmt.Sprintf("Error fetching playlists: %s", err.Error()))
		return err
	}
	if len(playlists) == 0 {
		_, err := m.Reply("❌ You don't have any playlists.")
		return err
	}
	var playlistInfo []string
	for _, playlist := range playlists {
		playlistInfo = append(playlistInfo, fmt.Sprintf("- %s (<code>%s</code>)", playlist.Name, playlist.ID))
	}
	_, err = m.Reply(fmt.Sprintf("<b>My Playlists</b>\n\n%s", strings.Join(playlistInfo, "\n")))
	return err
}
