/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/core/dl"
	"ashokshau/tgmusic/src/vc"
	"fmt"
	"html"
	"strings"

	"ashokshau/tgmusic/src/utils"

	td "github.com/AshokShau/gotdbot"
)

// playHandler handles the /play command.
func playHandler(c *td.Client, ctx *td.Context) error {
	if !playMode(c, ctx) {
		return td.EndGroups
	}

	return handlePlay(c, ctx, false)
}

// vPlayHandler handles the /vplay command.
func vPlayHandler(c *td.Client, ctx *td.Context) error {
	if !playMode(c, ctx) {
		return td.EndGroups
	}
	return handlePlay(c, ctx, true)
}

func handlePlay(c *td.Client, ctx *td.Context, isVideo bool) error {
	chatID := ctx.EffectiveChatId
	m := ctx.EffectiveMessage

	if queueLen := cache.ChatCache.GetQueueLength(chatID); queueLen > 10 {
		_, _ = m.ReplyText(c, "Queue is full (max 10 tracks). Use /end to clear.", nil)
		return td.EndGroups
	}

	isReply := m.ReplyToMessageID() != 0
	args := Args(m)
	url := getUrl(c, m, isReply)

	rMsg := m
	var err error
	if isReply && args == "" && url == "" {
		r, err := m.GetRepliedMessage(c)
		if err == nil && r != nil {
			args = r.Text()
		}
	}

	input := coalesce(url, args)

	if strings.HasPrefix(input, "tgpl_") {
		playlist, err := db.Instance.GetPlaylist(input)
		if err != nil {
			_, err = m.ReplyText(c, "❌ Playlist not found.", nil)
			return err
		}

		tracks := db.ConvertSongsToTracks(playlist.Songs)
		if len(tracks) == 0 {
			_, err = m.ReplyText(c, "❌ Playlist is empty.", nil)
			return err
		}

		updater, err := m.ReplyText(c, "🔍 Searching playlist...", nil)
		if err != nil {
			c.Logger.Warn("failed to send message", "error", err)
			return td.EndGroups
		}

		return handleMultipleTracks(c, m, updater, tracks, chatID, isVideo)
	}

	if match := utils.TelegramMessageRegex.FindStringSubmatch(input); match != nil {
		rMsg, err = utils.GetMessage(c, input)
		if err != nil {
			c.Logger.Warn("failed to parse message", "error", err.Error())
			_, err = m.ReplyText(c, "Invalid Telegram link.", nil)
			return err
		}
	} else if isReply {
		rMsg, err = m.GetRepliedMessage(c)
		if err != nil {
			_, err = m.ReplyText(c, "Invalid reply message.", nil)
			return err
		}
	}

	if isValid := isValidMedia(rMsg); isValid {
		isReply = true
	}

	if url == "" && args == "" && (!isReply || !isValidMedia(rMsg)) {
		_, _ = m.ReplyText(c, "<b>Usage:</b>\n/play [song or URL]\n\n<b>Supported Platforms:</b>\n- YouTube\n- Spotify\n- JioSaavn\n- Apple Music", &td.SendTextMessageOpts{ReplyMarkup: core.SupportKeyboard(), ParseMode: "HTML"})
		return td.EndGroups
	}

	updater, err := m.ReplyText(c, "🔍 Searching and downloading...", nil)
	if err != nil {
		c.Logger.Warn("failed to send message", "error", err)
		return td.EndGroups
	}

	if isReply && isValidMedia(rMsg) {
		return handleMedia(c, m, updater, rMsg, chatID, isVideo)
	}

	wrapper := dl.NewDownloaderWrapper(input)
	if url != "" {
		if !wrapper.IsValid() {
			_, _ = updater.EditText(c, "Invalid URL or unsupported platform.\n\n<b>Supported Platforms:</b>\n- YouTube\n- Spotify\n- JioSaavn\n- Apple Music", &td.EditTextMessageOpts{ReplyMarkup: core.SupportKeyboard(), ParseMode: "HTML"})
			return td.EndGroups
		}

		trackInfo, err := wrapper.GetInfo()
		if err != nil {
			_, _ = updater.EditText(c, fmt.Sprintf("❌ Error fetching track info: %s", err.Error()), nil)
			return td.EndGroups
		}

		if trackInfo.Results == nil || len(trackInfo.Results) == 0 {
			_, _ = updater.EditText(c, "No tracks found.", nil)
			return td.EndGroups
		}

		return handleUrl(c, m, updater, trackInfo, chatID, isVideo)
	}

	return handleTextSearch(c, m, updater, wrapper, chatID, isVideo)
}

// handleMedia handles playing media from a message.
func handleMedia(c *td.Client, m *td.Message, updater *td.Message, dlMsg *td.Message, chatId int64, isVideo bool) error {
	file, fileName := getFile(dlMsg)
	if file == nil {
		_, err := updater.EditText(c, "No valid media found in the message.", nil)
		return err
	}

	if file.Size > config.Conf.MaxFileSize {
		_, err := updater.EditText(c, fmt.Sprintf("File too large. Max size: %d MB.", config.Conf.MaxFileSize/(1024*1024)), nil)
		if err != nil {
			c.Logger.Warn("Edit message failed", "error", err)
		}
		return nil
	}

	fileId := dlMsg.RemoteFileID()
	if _track := cache.ChatCache.GetTrackIfExists(chatId, fileId); _track != nil {
		_, err := updater.EditText(c, "Track already in queue or playing.", nil)
		return err
	}

	dur := utils.GetFileDur(dlMsg)
	link, err := dlMsg.GetLink(c)
	if err != nil {
		c.Logger.Warn("Failed to get file link", "error", err)
		link.Link = ""
	}

	saveCache := utils.CachedTrack{
		URL: link.Link, Name: fileName, User: firstName(c, m), TrackID: fileId,
		Duration: dur, IsVideo: isVideo, Platform: utils.Telegram,
	}

	qLen := cache.ChatCache.AddSong(chatId, &saveCache)

	if qLen > 1 {
		queueInfo := fmt.Sprintf(
			"<u><b>Added to queue: %d</b></u>\n\n<b>Title:</b> <a href='%s'>%s</a>\n\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
			qLen, saveCache.URL, saveCache.Name, utils.SecToMin(saveCache.Duration), saveCache.User,
		)
		_, err := updater.EditText(c, queueInfo, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("play"), ParseMode: "HTML", DisableWebPagePreview: true})
		return err
	}

	file, err = dlMsg.Download(c, 1, 0, 0, true)
	if err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		_, err = updater.EditText(c, fmt.Sprintf("Download failed: %s", err.Error()), nil)
		return err
	}

	filePath := file.Local.Path
	if dur == 0 {
		dur = utils.GetMediaDuration(filePath)
		saveCache.Duration = dur
	}

	saveCache.FilePath = filePath

	if err = vc.Calls.PlayMedia(chatId, saveCache.FilePath, saveCache.IsVideo, ""); err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		_, err = updater.EditText(c, html.EscapeString(err.Error()), &td.EditTextMessageOpts{ParseMode: "HTML", DisableWebPagePreview: true})
		return err
	}

	nowPlaying := fmt.Sprintf(
		"<u><b>| Started streaming</b></u>\n\n<b>Title:</b> <a href='%s'>%s</a>\n\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
		saveCache.URL, saveCache.Name, utils.SecToMin(saveCache.Duration), saveCache.User,
	)

	_, err = updater.EditText(c, nowPlaying, &td.EditTextMessageOpts{
		ParseMode:             "HTML",
		ReplyMarkup:           core.ControlButtons("play"),
		DisableWebPagePreview: true,
	})

	return err
}

// handleTextSearch handles a text search for a song.
func handleTextSearch(c *td.Client, m *td.Message, updater *td.Message, wrapper *dl.DownloaderWrapper, chatId int64, isVideo bool) error {
	searchResult, err := wrapper.Search()
	if err != nil {
		_, err = updater.EditText(c, fmt.Sprintf("❌ Search failed: %s", err.Error()), nil)
		return err
	}

	if searchResult.Results == nil || len(searchResult.Results) == 0 {
		_, err = updater.EditText(c, "😕 No results found. Try a different query.", nil)
		return err
	}

	song := searchResult.Results[0]
	if _track := cache.ChatCache.GetTrackIfExists(chatId, song.Id); _track != nil {
		_, err := updater.EditText(c, "Track already in queue or playing.", nil)
		return err
	}

	return handleSingleTrack(c, m, updater, song, "", chatId, isVideo)
}

// handleUrl handles a URL search for a song.
func handleUrl(c *td.Client, m *td.Message, updater *td.Message, trackInfo utils.PlatformTracks, chatId int64, isVideo bool) error {
	if len(trackInfo.Results) == 1 {
		track := trackInfo.Results[0]
		if _track := cache.ChatCache.GetTrackIfExists(chatId, track.Id); _track != nil {
			_, err := updater.EditText(c, "Track already in queue or playing.", nil)
			return err
		}
		return handleSingleTrack(c, m, updater, track, "", chatId, isVideo)
	}

	return handleMultipleTracks(c, m, updater, trackInfo.Results, chatId, isVideo)
}

// handleSingleTrack handles a single track.
func handleSingleTrack(c *td.Client, m *td.Message, updater *td.Message, song utils.MusicTrack, filePath string, chatId int64, isVideo bool) error {
	if song.Duration > int(config.Conf.SongDurationLimit) {
		_, err := updater.EditText(c, fmt.Sprintf("Sorry, song exceeds max duration of %d minutes.", config.Conf.SongDurationLimit/60), nil)
		return err
	}

	saveCache := utils.CachedTrack{
		URL: song.Url, Name: song.Title, User: firstName(c, m), FilePath: filePath,
		Thumbnail: song.Thumbnail, TrackID: song.Id, Duration: song.Duration, Channel: song.Channel, Views: song.Views,
		IsVideo: isVideo, Platform: song.Platform,
	}

	qLen := cache.ChatCache.AddSong(chatId, &saveCache)

	if qLen > 1 {
		queueInfo := fmt.Sprintf(
			"<u><b>Added to queue: %d</b></u>\n\n<b>Title:</b> <a href='%s'>%s</a>\n\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
			qLen, saveCache.URL, saveCache.Name, utils.SecToMin(saveCache.Duration), saveCache.User,
		)

		_, err := updater.EditText(c, queueInfo, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("play"), ParseMode: "HTML", DisableWebPagePreview: true})
		return err
	}

	if saveCache.FilePath == "" {
		dlResult, err := dl.DownloadCachedTrack(&saveCache, c)
		if err != nil {
			cache.ChatCache.RemoveCurrentSong(chatId)
			_, err = updater.EditText(c, fmt.Sprintf("Download failed: %s", err.Error()), nil)
			return err
		}

		saveCache.FilePath = dlResult
	}

	if err := vc.Calls.PlayMedia(chatId, saveCache.FilePath, saveCache.IsVideo, ""); err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		_, err = updater.EditText(c, err.Error(), &td.EditTextMessageOpts{ParseMode: "HTML", DisableWebPagePreview: true})
		return err
	}

	nowPlaying := fmt.Sprintf(
		"<u><b>| Started streaming</b></u>\n\n<b>Title:</b> <a href='%s'>%s</a>\n\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
		saveCache.URL, saveCache.Name, utils.SecToMin(song.Duration), saveCache.User,
	)

	_, err := updater.EditText(c, nowPlaying, &td.EditTextMessageOpts{
		ReplyMarkup:           core.ControlButtons("play"),
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})

	if err != nil {
		c.Logger.Warn("Edit message failed", "error", err)
		return err
	}

	return nil
}

// handleMultipleTracks handles multiple tracks.
func handleMultipleTracks(c *td.Client, m *td.Message, updater *td.Message, tracks []utils.MusicTrack, chatId int64, isVideo bool) error {
	if len(tracks) == 0 {
		_, err := updater.EditText(c, "No tracks found.", nil)
		return err
	}

	queueHeader := "<u><b>Added to Queue:</b></u>\n<blockquote expandable>\n"
	var tracksToAdd []*utils.CachedTrack
	var skippedTracks []string

	shouldPlayFirst := false
	var firstTrack *utils.CachedTrack

	for _, track := range tracks {
		if track.Duration > int(config.Conf.SongDurationLimit) {
			skippedTracks = append(skippedTracks, track.Title)
			continue
		}

		saveCache := &utils.CachedTrack{
			Name: track.Title, TrackID: track.Id, Duration: track.Duration,
			Thumbnail: track.Thumbnail, User: firstName(c, m), Platform: track.Platform,
			IsVideo: isVideo, URL: track.Url, Channel: track.Channel, Views: track.Views,
		}
		tracksToAdd = append(tracksToAdd, saveCache)
	}

	if len(tracksToAdd) == 0 {
		if len(skippedTracks) > 0 {
			_, err := updater.EditText(c, fmt.Sprintf("All tracks were skipped (max duration %d min).", config.Conf.SongDurationLimit/60), nil)
			return err
		}
		_, err := updater.EditText(c, "No valid tracks found.", nil)
		return err
	}

	qLenAfter := cache.ChatCache.AddSongs(chatId, tracksToAdd)
	startLen := qLenAfter - len(tracksToAdd)

	if startLen == 0 {
		shouldPlayFirst = true
		firstTrack = tracksToAdd[0]
		firstTrack.Loop = 1
	}

	var sb strings.Builder
	sb.WriteString(queueHeader)

	totalDuration := 0
	for i, track := range tracksToAdd {
		currentQLen := startLen + i + 1
		fmt.Fprintf(&sb, "<b>%d.</b> %s\n└ Duration: %s\n",
			currentQLen, track.Name, utils.SecToMin(track.Duration))
		totalDuration += track.Duration
	}

	sb.WriteString("</blockquote>")
	queueSummary := fmt.Sprintf(
		"\n<b>Queue Total:</b> %d\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
		qLenAfter, utils.SecToMin(totalDuration), firstName(c, m),
	)

	sb.WriteString(queueSummary)
	if len(skippedTracks) > 0 {
		fmt.Fprintf(&sb, "\n\n<b>Skipped %d tracks</b> (exceeded duration limit).", len(skippedTracks))
	}

	fullMessage := sb.String()

	if len(fullMessage) > 4096 {
		fullMessage = queueSummary
	}

	if shouldPlayFirst && firstTrack != nil {
		_ = vc.Calls.PlayNext(chatId)
	}

	_, err := updater.EditText(c, fullMessage, &td.EditTextMessageOpts{
		ParseMode:             "HTML",
		ReplyMarkup:           core.ControlButtons("play"),
		DisableWebPagePreview: true,
	})

	return err
}
