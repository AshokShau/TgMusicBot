/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/core/dl"
	"ashokshau/tgmusic/src/vc"

	"ashokshau/tgmusic/src/utils"

	"github.com/amarnathcjd/gogram/telegram"
)

// playHandler handles the /play command.
func playHandler(m *telegram.NewMessage) error {
	return handlePlay(m, false)
}

// vPlayHandler handles the /vplay command.
func vPlayHandler(m *telegram.NewMessage) error {
	return handlePlay(m, true)
}

func handlePlay(m *telegram.NewMessage, isVideo bool) error {
	chatID := m.ChannelID()
	if queueLen := cache.ChatCache.GetQueueLength(chatID); queueLen > 10 {
		_, _ = m.Reply("⚠️ Queue is full (max 10 tracks). Use /end to clear.")
		return telegram.ErrEndGroup
	}

	isReply := m.IsReply()
	url := getUrl(m, isReply)
	args := m.Args()
	rMsg := m
	var err error

	input := coalesce(url, args)

	if strings.HasPrefix(input, "tgpl_") {
		ctx, cancel := db.Ctx()
		defer cancel()
		playlist, err := db.Instance.GetPlaylist(ctx, input)
		if err != nil {
			_, err = m.Reply("❌ Playlist not found.")
			return err
		}

		tracks := db.ConvertSongsToTracks(playlist.Songs)
		if len(tracks) == 0 {
			_, err = m.Reply("❌ Playlist is empty.")
			return err
		}

		updater, err := m.Reply("🔍 Searching playlist...")
		if err != nil {
			logger.Warn("failed to send message: %v", err)
			return telegram.ErrEndGroup
		}

		return handleMultipleTracks(m, updater, tracks, chatID, isVideo)
	}

	if username, msgID, ok := parseTelegramURL(input); ok {
		rMsg, err = m.Client.GetMessageByID(username, int32(msgID))
		if err != nil {
			_, err = m.Reply("❌ Invalid Telegram link.")
			return err
		}
	} else if isReply {
		rMsg, err = m.GetReplyMessage()
		if err != nil {
			_, err = m.Reply("❌ Invalid reply message.")
			return err
		}
	}

	if isValid := isValidMedia(rMsg); isValid {
		isReply = true
	}

	if url == "" && args == "" && (!isReply || !isValidMedia(rMsg)) {
		_, _ = m.Reply("🎵 <b>Usage:</b>\n/play [song or URL]\n\n<b>Supported Platforms:</b>\n- YouTube\n- Spotify\n- JioSaavn\n- Apple Music", &telegram.SendOptions{ReplyMarkup: core.SupportKeyboard()})
		return telegram.ErrEndGroup
	}

	updater, err := m.Reply("🔍 Searching...")
	if err != nil {
		logger.Warn("failed to send message: %v", err)
		return telegram.ErrEndGroup
	}

	if isReply && isValidMedia(rMsg) {
		return handleMedia(m, updater, rMsg, chatID, isVideo)
	}

	wrapper := dl.NewDownloaderWrapper(input)
	if url != "" {
		if !wrapper.IsValid() {
			_, _ = updater.Edit("❌ Invalid URL or unsupported platform.\n\n<b>Supported Platforms:</b>\n- YouTube\n- Spotify\n- JioSaavn\n- Apple Music", &telegram.SendOptions{ReplyMarkup: core.SupportKeyboard()})
			return telegram.ErrEndGroup
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		trackInfo, err := wrapper.GetInfo(ctx)
		if err != nil {
			_, _ = updater.Edit(fmt.Sprintf("❌ Error fetching track info: %s", err.Error()))
			return telegram.ErrEndGroup
		}

		if trackInfo.Results == nil || len(trackInfo.Results) == 0 {
			_, _ = updater.Edit("❌ No tracks found.")
			return telegram.ErrEndGroup
		}

		return handleUrl(m, updater, trackInfo, chatID, isVideo)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()
	return handleTextSearch(m, updater, wrapper, chatID, isVideo, ctx2)
}

// handleMedia handles playing media from a message.
func handleMedia(m *telegram.NewMessage, updater *telegram.NewMessage, dlMsg *telegram.NewMessage, chatId int64, isVideo bool) error {
	if dlMsg.File.Size > config.Conf.MaxFileSize {
		_, err := updater.Edit(fmt.Sprintf("❌ File too large. Max size: %d MB.", config.Conf.MaxFileSize/(1024*1024)))
		if err != nil {
			logger.Warn("Edit message failed: %v", err)
		}
		return nil
	}

	fileName := dlMsg.File.Name
	fileId := dlMsg.File.FileID
	if _track := cache.ChatCache.GetTrackIfExists(chatId, fileId); _track != nil {
		_, err := updater.Edit("✅ Track already in queue or playing.")
		return err
	}

	dur := utils.GetFileDur(dlMsg)
	saveCache := utils.CachedTrack{
		URL: dlMsg.Link(), Name: fileName, User: m.Sender.FirstName, TrackID: fileId,
		Duration: dur, IsVideo: isVideo, Platform: utils.Telegram,
	}

	qLen := cache.ChatCache.AddSong(chatId, &saveCache)

	if qLen > 1 {
		queueInfo := fmt.Sprintf(
			"<b>🎧 Added to Queue (#%d)</b>\n\n<b>Track:</b> <a href='%s'>%s</a>\n<b>Duration:</b> %s\n<b>By:</b> %s",
			qLen, saveCache.URL, saveCache.Name, utils.SecToMin(saveCache.Duration), saveCache.User,
		)
		_, err := updater.Edit(queueInfo, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("play")})
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	filePath, err := dlMsg.Download(&telegram.DownloadOptions{FileName: filepath.Join(config.Conf.DownloadsDir, fileName), Ctx: ctx})
	if err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId) // Cleanup on failure
		_, err = updater.Edit(fmt.Sprintf("❌ Download failed: %s", err.Error()))
		return err
	}

	if dur == 0 {
		dur = utils.GetMediaDuration(filePath)
		saveCache.Duration = dur
	}

	saveCache.FilePath = filePath
	if err := vc.Calls.PlayMedia(chatId, saveCache.FilePath, saveCache.IsVideo, ""); err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		_, err = updater.Edit(err.Error())
		return err
	}

	nowPlaying := fmt.Sprintf(
		"🎵 <b>Now Playing:</b>\n\n<b>Track:</b> <a href='%s'>%s</a>\n<b>Duration:</b> %s\n<b>By:</b> %s",
		saveCache.URL, saveCache.Name, utils.SecToMin(saveCache.Duration), saveCache.User,
	)

	_, err = updater.Edit(nowPlaying, &telegram.SendOptions{
		ReplyMarkup: core.ControlButtons("play"),
	})
	return err
}

// handleTextSearch handles a text search for a song.
func handleTextSearch(m *telegram.NewMessage, updater *telegram.NewMessage, wrapper *dl.DownloaderWrapper, chatId int64, isVideo bool, ctx context.Context) error {
	searchResult, err := wrapper.Search(ctx)
	if err != nil {
		_, err = updater.Edit(fmt.Sprintf("❌ Search failed: %s", err.Error()))
		return err
	}

	if searchResult.Results == nil || len(searchResult.Results) == 0 {
		_, err = updater.Edit("😕 No results found. Try a different query.")
		return err
	}

	song := searchResult.Results[0]
	if _track := cache.ChatCache.GetTrackIfExists(chatId, song.Id); _track != nil {
		_, err := updater.Edit("✅ Track already in queue or playing.")
		return err
	}

	return handleSingleTrack(m, updater, song, "", chatId, isVideo)
}

// handleUrl handles a URL search for a song.
func handleUrl(m *telegram.NewMessage, updater *telegram.NewMessage, trackInfo utils.PlatformTracks, chatId int64, isVideo bool) error {
	if len(trackInfo.Results) == 1 {
		track := trackInfo.Results[0]
		if _track := cache.ChatCache.GetTrackIfExists(chatId, track.Id); _track != nil {
			_, err := updater.Edit("✅ Track already in queue or playing.")
			return err
		}
		return handleSingleTrack(m, updater, track, "", chatId, isVideo)
	}

	return handleMultipleTracks(m, updater, trackInfo.Results, chatId, isVideo)
}

// handleSingleTrack handles a single track.
func handleSingleTrack(m *telegram.NewMessage, updater *telegram.NewMessage, song utils.MusicTrack, filePath string, chatId int64, isVideo bool) error {
	if song.Duration > int(config.Conf.SongDurationLimit) {
		_, err := updater.Edit(fmt.Sprintf("Sorry, song exceeds max duration of %d minutes.", config.Conf.SongDurationLimit/60))
		return err
	}

	saveCache := utils.CachedTrack{
		URL: song.Url, Name: song.Title, User: m.Sender.FirstName, FilePath: filePath,
		Thumbnail: song.Thumbnail, TrackID: song.Id, Duration: song.Duration, Channel: song.Channel, Views: song.Views,
		IsVideo: isVideo, Platform: song.Platform,
	}

	qLen := cache.ChatCache.AddSong(chatId, &saveCache)

	if qLen > 1 {
		queueInfo := fmt.Sprintf(
			"<b>🎧 Added to Queue (#%d)</b>\n\n<b>Track:</b> <a href='%s'>%s</a>\n<b>Duration:</b> %s\n<b>By:</b> %s",
			qLen, saveCache.URL, saveCache.Name, utils.SecToMin(saveCache.Duration), saveCache.User,
		)

		_, err := updater.Edit(queueInfo, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("play")})
		return err
	}

	if saveCache.FilePath == "" {
		_, err := updater.Edit(fmt.Sprintf("Downloading %s...", song.Title))
		if err != nil {
			logger.Warn("Edit message failed: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		dlResult, err := dl.DownloadSong(ctx, &saveCache, m.Client)
		if err != nil {
			cache.ChatCache.RemoveCurrentSong(chatId)
			_, err = updater.Edit(fmt.Sprintf("❌ Download failed: %s", err.Error()))
			return err
		}

		saveCache.FilePath = dlResult
	}

	if err := vc.Calls.PlayMedia(chatId, saveCache.FilePath, saveCache.IsVideo, ""); err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		_, err = updater.Edit(err.Error())
		return err
	}

	nowPlaying := fmt.Sprintf(
		"🎵 <b>Now Playing:</b>\n\n<b>Track:</b> <a href='%s'>%s</a>\n<b>Duration:</b> %s\n<b>By:</b> %s",
		saveCache.URL, saveCache.Name, utils.SecToMin(song.Duration), saveCache.User,
	)

	_, err := updater.Edit(nowPlaying, &telegram.SendOptions{
		ReplyMarkup: core.ControlButtons("play"),
	})

	if err != nil {
		logger.Warn("Edit message failed: %v", err)
		return err
	}

	return nil
}

// handleMultipleTracks handles multiple tracks.
func handleMultipleTracks(m *telegram.NewMessage, updater *telegram.NewMessage, tracks []utils.MusicTrack, chatId int64, isVideo bool) error {
	if len(tracks) == 0 {
		_, err := updater.Edit("❌ No tracks found.")
		return err
	}

	queueHeader := "<b>📥 Added to Queue:</b>\n<blockquote collapsed='true'>\n"
	var queueItems []string
	var skippedTracks []string

	shouldPlayFirst := false
	var firstTrack *utils.CachedTrack

	for i, track := range tracks {
		if track.Duration > int(config.Conf.SongDurationLimit) {
			skippedTracks = append(skippedTracks, track.Title)
			continue
		}

		saveCache := utils.CachedTrack{
			Name: track.Title, TrackID: track.Id, Duration: track.Duration,
			Thumbnail: track.Thumbnail, User: m.Sender.FirstName, Platform: track.Platform,
			IsVideo: isVideo, URL: track.Url, Channel: track.Channel, Views: track.Views,
		}

		qLen := cache.ChatCache.AddSong(chatId, &saveCache)
		if i == 0 && qLen == 1 {
			shouldPlayFirst = true
			firstTrack = &saveCache
			saveCache.Loop = 1
		}

		queueItems = append(queueItems,
			fmt.Sprintf("<b>%d.</b> %s\n└ Duration: %s",
				qLen, track.Title, utils.SecToMin(track.Duration)),
		)
	}

	totalDuration := 0
	for _, t := range tracks {
		totalDuration += t.Duration
	}

	queueSummary := fmt.Sprintf(
		"</blockquote>\n<b>📋 Queue Total:</b> %d\n<b>⏱ Duration:</b> %s\n<b>👤 By:</b> %s",
		cache.ChatCache.GetQueueLength(chatId), utils.SecToMin(totalDuration), m.Sender.FirstName,
	)

	fullMessage := queueHeader + strings.Join(queueItems, "\n") + queueSummary
	if len(skippedTracks) > 0 {
		fullMessage += fmt.Sprintf("\n\n<b>Skipped %d tracks</b> (exceeded duration limit).", len(skippedTracks))
	}

	if len(fullMessage) > 4096 {
		fullMessage = queueSummary
	}

	if shouldPlayFirst && firstTrack != nil {
		_ = vc.Calls.PlayNext(chatId)
		/*
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()

				dlPath, err := dl.DownloadSong(ctx, firstTrack, m.Client)
				if err != nil {
					cache.ChatCache.RemoveCurrentSong(chatId)
					logger.Warn("failed to download song: %v", err)
					_, _ = m.Client.SendMessage(chatId, "failed to download song")
					return
				}

				firstTrack.FilePath = dlPath
				if err := vc.Calls.PlayMedia(chatId, firstTrack.FilePath, firstTrack.IsVideo, ""); err != nil {
					cache.ChatCache.RemoveCurrentSong(chatId)
					return
				}

				nowPlaying := fmt.Sprintf(
					"🎵 <b>Now Playing:</b>\n\n▫ <b>Track:</b> <a href='%s'>%s</a>\n▫ <b>Duration:</b> %s\n▫ <b>Requested by:</b> %s",
					firstTrack.URL, firstTrack.Name, utils.SecToMin(firstTrack.Duration), firstTrack.User,
				)
				_, _ = m.Client.SendMessage(chatId, nowPlaying, &telegram.SendOptions{ReplyMarkup: core.ControlButtons("play")})
			}()

		*/
	}

	_, err := updater.Edit(fullMessage, &telegram.SendOptions{
		ReplyMarkup: core.ControlButtons("play"),
	})

	return err
}
