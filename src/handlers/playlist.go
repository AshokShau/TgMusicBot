/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
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

	td "github.com/AshokShau/gotdbot"
)

func createPlaylistHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage
	userID := m.SenderID()

	args := Args(m)
	if args == "" {
		_, err := m.ReplyText(c, "<b>Usage:</b> /createplaylist [playlist name]", replyOpts)
		return err
	}

	userPlaylists, err := db.Instance.GetUserPlaylists(userID)
	if err != nil {
		_, err = m.ReplyText(c, "Unable to fetch your playlists. Please try again later.", nil)
		return err
	}

	if len(userPlaylists) >= 10 {
		_, _ = m.ReplyText(c, "You have reached the maximum limit of 10 playlists.", nil)
		return td.EndGroups
	}

	if len([]rune(args)) > 40 {
		args = string([]rune(args)[:40])
	}

	playlistID, err := db.Instance.CreatePlaylist(args, userID)
	if err != nil {
		_, err = m.ReplyText(c, fmt.Sprintf("Failed to create playlist: %s", err.Error()), nil)
		return err
	}

	_, err = m.ReplyText(
		c,
		fmt.Sprintf(
			"Playlist <b>%s</b> has been created successfully.\nID: <code>%s</code>",
			args,
			playlistID,
		),
		replyOpts,
	)

	return td.EndGroups
}

func deletePlaylistHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage
	userID := m.SenderID()

	args := Args(m)
	if args == "" {
		_, err := m.ReplyText(
			c,
			"<b>Usage:</b> /deleteplaylist [playlist id]",
			&td.SendTextMessageOpts{ParseMode: "HTML"},
		)
		return err
	}

	playlist, err := db.Instance.GetPlaylist(args)
	if err != nil {
		_, err := m.ReplyText(
			c,
			"The specified playlist could not be found. Please check the playlist ID.",
			nil,
		)
		return err
	}

	if playlist.UserID != userID {
		_, err := m.ReplyText(
			c,
			"You can only delete playlists that you created.",
			nil,
		)
		return err
	}

	err = db.Instance.DeletePlaylist(args, userID)
	if err != nil {
		_, err := m.ReplyText(
			c,
			fmt.Sprintf("Failed to delete the playlist: %s", err.Error()),
			nil,
		)
		return err
	}

	_, err = m.ReplyText(
		c,
		fmt.Sprintf("Playlist <b>%s</b> has been deleted successfully.", playlist.Name),
		&td.SendTextMessageOpts{ParseMode: "HTML"},
	)

	return err
}
func addToPlaylistHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage
	userID := m.SenderID()

	args := strings.SplitN(Args(m), " ", 2)
	if len(args) != 2 {
		_, err := m.ReplyText(
			c,
			"<b>Usage:</b> /addtoplaylist [playlist id] [song url]",
			&td.SendTextMessageOpts{ParseMode: "HTML"},
		)
		return err
	}

	playlistID := args[0]
	songURL := args[1]

	playlist, err := db.Instance.GetPlaylist(playlistID)
	if err != nil {
		_, err := m.ReplyText(
			c,
			"The specified playlist could not be found. Please verify the playlist ID.",
			nil,
		)
		return err
	}

	if playlist.UserID != userID {
		_, err := m.ReplyText(
			c,
			"You can only modify playlists that you created.",
			nil,
		)
		return err
	}

	wrapper := dl.NewDownloaderWrapper(songURL)
	if !wrapper.IsValid() {
		_, err := m.ReplyText(
			c,
			"The provided URL is invalid or the platform is not supported.",
			nil,
		)
		return err
	}

	trackInfo, err := wrapper.GetInfo()
	if err != nil {
		_, err := m.ReplyText(
			c,
			fmt.Sprintf("Unable to retrieve track information: %s", err.Error()),
			nil,
		)
		return err
	}

	if trackInfo.Results == nil || len(trackInfo.Results) == 0 {
		_, err := m.ReplyText(
			c,
			"No playable tracks were found for the provided link.",
			nil,
		)
		return err
	}

	song := db.Song{
		URL:      trackInfo.Results[0].Url,
		Name:     trackInfo.Results[0].Title,
		TrackID:  trackInfo.Results[0].Id,
		Duration: trackInfo.Results[0].Duration,
		Platform: trackInfo.Results[0].Platform,
	}

	err = db.Instance.AddSongToPlaylist(playlistID, song)
	if err != nil {
		_, err := m.ReplyText(
			c,
			fmt.Sprintf("Failed to add the track to the playlist: %s", err.Error()),
			nil,
		)
		return err
	}

	_, err = m.ReplyText(
		c,
		fmt.Sprintf(
			"Track <b>%s</b> has been added to playlist <b>%s</b>.",
			song.Name,
			playlist.Name,
		),
		replyOpts,
	)

	return err
}

func removeFromPlaylistHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage
	userID := m.SenderID()

	args := strings.SplitN(Args(m), " ", 2)
	if len(args) != 2 {
		_, err := m.ReplyText(
			c,
			"<b>Usage:</b> /removefromplaylist [playlist id] [song number or url]",
			&td.SendTextMessageOpts{ParseMode: "HTML"},
		)
		return err
	}

	playlistID := args[0]
	songIdentifier := args[1]

	playlist, err := db.Instance.GetPlaylist(playlistID)
	if err != nil {
		_, err = m.ReplyText(c, "Playlist not found.", nil)
		return err
	}

	if playlist.UserID != userID {
		_, err = m.ReplyText(c, "You do not own this playlist.", nil)
		return err
	}

	songIndex, err := strconv.Atoi(songIdentifier)
	var trackID string

	if err == nil {
		if songIndex < 1 || songIndex > len(playlist.Songs) {
			_, err := m.ReplyText(c, "Invalid song number.", nil)
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
		_, err = m.ReplyText(c, "Song not found in playlist.", nil)
		return err
	}

	err = db.Instance.RemoveSongFromPlaylist(playlistID, trackID)
	if err != nil {
		_, err = m.ReplyText(c, fmt.Sprintf("Error removing song: %s", err.Error()), nil)
		return err
	}

	_, err = m.ReplyText(c, fmt.Sprintf("Song removed from playlist '%s'.", playlist.Name), nil)
	return err
}

func playlistInfoHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage

	args := Args(m)
	if args == "" {
		_, err := m.ReplyText(
			c,
			"<b>Usage:</b> /playlistinfo [playlist id]",
			&td.SendTextMessageOpts{ParseMode: "HTML"},
		)
		return err
	}

	playlist, err := db.Instance.GetPlaylist(args)
	if err != nil {
		_, err = m.ReplyText(c, "Playlist not found.", nil)
		return err
	}

	var songs []string
	for i, song := range playlist.Songs {
		songs = append(songs, fmt.Sprintf("%d. %s (%s)", i+1, song.Name, song.URL))
	}

	owner, err := c.GetUser(playlist.UserID)
	if err != nil {
		return td.EndGroups
	}

	_, err = m.ReplyText(
		c,
		fmt.Sprintf(
			"<b>Playlist Info</b>\n\n<b>Name:</b> %s\n<b>Owner:</b> %s\n<b>Songs:</b> %d\n\n%s",
			playlist.Name,
			owner.FirstName,
			len(playlist.Songs),
			strings.Join(songs, "\n"),
		),
		&td.SendTextMessageOpts{ParseMode: "HTML"},
	)
	return td.EndGroups
}

func myPlaylistsHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage
	userID := m.SenderID()

	playlists, err := db.Instance.GetUserPlaylists(userID)
	if err != nil {
		_, err := m.ReplyText(c, fmt.Sprintf("Error fetching playlists: %s", err.Error()), nil)
		return err
	}

	if len(playlists) == 0 {
		_, err := m.ReplyText(c, "You do not have any playlists.", nil)
		return err
	}

	var playlistInfo []string
	for _, playlist := range playlists {
		playlistInfo = append(
			playlistInfo,
			fmt.Sprintf("- %s (<code>%s</code>)", playlist.Name, playlist.ID),
		)
	}

	_, err = m.ReplyText(
		c,
		fmt.Sprintf("<b>My Playlists</b>\n\n%s", strings.Join(playlistInfo, "\n")),
		&td.SendTextMessageOpts{ParseMode: "HTML"},
	)

	return err
}
