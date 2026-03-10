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
	ctx2, cancel := db.Ctx()
	defer cancel()

	args := Args(m)
	if args == "" {
		_, err := m.ReplyText(c, "<b>Usage:</b> /createplaylist [playlist name]", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	userPlaylists, err := db.Instance.GetUserPlaylists(ctx2, userID)
	if err != nil {
		_, err := m.ReplyText(c, "Error creating playlist.", nil)
		return err
	}

	if len(userPlaylists) >= 10 {
		_, _ = m.ReplyText(c, fmt.Sprintf("You have reached the limit of %d playlists.", 10), nil)
		return td.EndGroups
	}

	if len([]rune(args)) > 40 {
		args = string([]rune(args)[:40])
	}

	playlistID, err := db.Instance.CreatePlaylist(ctx2, args, userID)
	if err != nil {
		_, err := m.ReplyText(c, fmt.Sprintf("Error creating playlist: %s", err.Error()), nil)
		return err
	}

	_, err = m.ReplyText(c, fmt.Sprintf("✅ Playlist '%s' created (ID: <code>%s</code>).", args, playlistID), &td.SendTextMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}

func deletePlaylistHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage
	userID := m.SenderID()
	ctx2, cancel := db.Ctx()
	defer cancel()
	args := Args(m)
	if args == "" {
		_, err := m.ReplyText(c, "<b>Usage:</b> /deleteplaylist [playlist id]", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}
	playlist, err := db.Instance.GetPlaylist(ctx2, args)
	if err != nil {
		_, err := m.ReplyText(c, "❌ Playlist not found.", nil)
		return err
	}
	if playlist.UserID != userID {
		_, err := m.ReplyText(c, "❌ You don't own this playlist.", nil)
		return err
	}

	err = db.Instance.DeletePlaylist(ctx2, args, userID)
	if err != nil {
		_, err := m.ReplyText(c, fmt.Sprintf("Error deleting playlist: %s", err.Error()), nil)
		return err
	}

	_, err = m.ReplyText(c, fmt.Sprintf("✅ Playlist '%s' deleted.", playlist.Name), nil)
	return err
}

func addToPlaylistHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage
	userID := m.SenderID()
	ctx2, cancel := db.Ctx()
	defer cancel()

	args := strings.SplitN(Args(m), " ", 2)
	if len(args) != 2 {
		_, err := m.ReplyText(c, "<b>Usage:</b> /addtoplaylist [playlist id] [song url]", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}
	playlistID := args[0]
	songURL := args[1]
	playlist, err := db.Instance.GetPlaylist(ctx2, playlistID)
	if err != nil {
		_, err := m.ReplyText(c, "❌ Playlist not found.", nil)
		return err
	}
	if playlist.UserID != userID {
		_, err := m.ReplyText(c, "❌ You don't own this playlist.", nil)
		return err
	}
	wrapper := dl.NewDownloaderWrapper(songURL)
	if !wrapper.IsValid() {
		_, err := m.ReplyText(c, "❌ Invalid URL or unsupported platform.", nil)
		return err
	}
	trackInfo, err := wrapper.GetInfo(ctx2)
	if err != nil {
		_, err := m.ReplyText(c, fmt.Sprintf("❌ Error fetching track info: %s", err.Error()), nil)
		return err
	}

	if trackInfo.Results == nil {
		_, err := m.ReplyText(c, "❌ No tracks found.", nil)
		return err
	}

	song := db.Song{
		URL:      trackInfo.Results[0].Url,
		Name:     trackInfo.Results[0].Title,
		TrackID:  trackInfo.Results[0].Id,
		Duration: trackInfo.Results[0].Duration,
		Platform: trackInfo.Results[0].Platform,
	}

	err = db.Instance.AddSongToPlaylist(ctx2, playlistID, song)
	if err != nil {
		_, err := m.ReplyText(c, fmt.Sprintf("Error adding song: %s", err.Error()), nil)
		return err
	}
	_, err = m.ReplyText(c, fmt.Sprintf("✅ '%s' added to playlist '%s'.", song.Name, playlist.Name), nil)
	return err
}

func removeFromPlaylistHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage
	userID := m.SenderID()
	ctx2, cancel := db.Ctx()
	defer cancel()
	args := strings.SplitN(Args(m), " ", 2)
	if len(args) != 2 {
		_, err := m.ReplyText(c, "<b>Usage:</b> /removefromplaylist [playlist id] [song number or url]", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}
	playlistID := args[0]
	songIdentifier := args[1]
	playlist, err := db.Instance.GetPlaylist(ctx2, playlistID)
	if err != nil {
		_, err = m.ReplyText(c, "❌ Playlist not found.", nil)
		return err
	}

	if playlist.UserID != userID {
		_, err = m.ReplyText(c, "❌ You don't own this playlist.", nil)
		return err
	}

	songIndex, err := strconv.Atoi(songIdentifier)
	var trackID string
	if err == nil {
		if songIndex < 1 || songIndex > len(playlist.Songs) {
			_, err := m.ReplyText(c, "❌ Invalid song number.", nil)
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
		_, err = m.ReplyText(c, "❌ Song not found in playlist.", nil)
		return err
	}

	c.Logger.Info("Removing song from playlist", "id", playlistID, "id", trackID)
	err = db.Instance.RemoveSongFromPlaylist(ctx2, playlistID, trackID)
	if err != nil {
		_, err = m.ReplyText(c, fmt.Sprintf("Error removing song: %s", err.Error()), nil)
		return err
	}

	_, err = m.ReplyText(c, fmt.Sprintf("✅ Song removed from playlist '%s'.", playlist.Name), nil)
	return err
}

func playlistInfoHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage
	ctx2, cancel := db.Ctx()
	defer cancel()
	args := Args(m)
	if args == "" {
		_, err := m.ReplyText(c, "<b>Usage:</b> /playlistinfo [playlist id]", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	playlist, err := db.Instance.GetPlaylist(ctx2, args)
	if err != nil {
		_, err = m.ReplyText(c, "❌ Playlist not found.", nil)
		return err
	}

	var songs []string
	for i, song := range playlist.Songs {
		songs = append(songs, fmt.Sprintf("%d. %s (%s)", i+1, song.Name, song.URL))
	}

	owner, err := c.GetUser(playlist.UserID)
	if err != nil {
		c.Logger.Warn(err.Error())
		return td.EndGroups
	}

	_, err = m.ReplyText(c, fmt.Sprintf("<b>Playlist Info</b>\n\n<b>Name:</b> %s\n<b>Owner:</b> %s\n<b>Songs:</b> %d\n\n%s", playlist.Name, owner.FirstName, len(playlist.Songs), strings.Join(songs, "\n")), &td.SendTextMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}

func myPlaylistsHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage

	userID := m.SenderID()
	ctx2, cancel := db.Ctx()
	defer cancel()
	playlists, err := db.Instance.GetUserPlaylists(ctx2, userID)
	if err != nil {
		_, err := m.ReplyText(c, fmt.Sprintf("Error fetching playlists: %s", err.Error()), nil)
		return err
	}
	if len(playlists) == 0 {
		_, err := m.ReplyText(c, "❌ You don't have any playlists.", nil)
		return err
	}
	var playlistInfo []string
	for _, playlist := range playlists {
		playlistInfo = append(playlistInfo, fmt.Sprintf("- %s (<code>%s</code>)", playlist.Name, playlist.ID))
	}
	_, err = m.ReplyText(c, fmt.Sprintf("<b>My Playlists</b>\n\n%s", strings.Join(playlistInfo, "\n")), &td.SendTextMessageOpts{ParseMode: "HTML"})
	return err
}
