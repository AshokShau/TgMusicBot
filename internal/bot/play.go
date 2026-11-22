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
	"ashokshau/tgmusic/internal/config"
	"ashokshau/tgmusic/internal/db"
	"ashokshau/tgmusic/internal/downloader"
	"ashokshau/tgmusic/internal/utils"
	"fmt"
	"html"
	"slices"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

// playHandler handles the /play command.
func playHandler(c *td.Client, m *td.Message) error {
	if !playMode(c, m) {
		return td.EndGroups
	}

	return handlePlay(c, m, false, false)
}

// vPlayHandler handles the /vplay command.
func vPlayHandler(c *td.Client, m *td.Message) error {
	if !playMode(c, m) {
		return td.EndGroups
	}

	if !config.EnableVideoPlayback {
		_, _ = m.ReplyText(c, "🎥 Video playback is currently disabled.\n\nAs more people use the bot, video streaming can sometimes cause lag and reduce music quality in voice chats. To ensure a smooth listening experience for everyone, this feature has been turned off for now.\n\nThanks for your support and understanding ❤️", nil)
		return td.EndGroups
	}
	return handlePlay(c, m, true, false)
}

// fPlayHandler handles the /fplay command.
func fPlayHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	return handlePlay(c, m, false, true)
}

// fVPlayHandler handles the /fvplay command.
func fVPlayHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	if !config.EnableVideoPlayback {
		_, _ = m.ReplyText(c, "🎥 Video playback is currently disabled.\n\nAs more people use the bot, video streaming can sometimes cause lag and reduce music quality in voice chats. To ensure a smooth listening experience for everyone, this feature has been turned off for now.\n\nThanks for your support and understanding ❤️", nil)
		return td.EndGroups
	}
	return handlePlay(c, m, true, true)
}

func handlePlay(c *td.Client, m *td.Message, isVideo bool, force bool) error {
	chatID := m.ChatId

	if queueLen := cache.ChatCache.GetQueueLength(chatID); queueLen > 10 {
		_, _ = m.ReplyText(c, "Queue is full (max 10 tracks). Use /end to clear.", nil)
		return td.EndGroups
	}

	isReply := m.ReplyToMessageID() != 0
	args := Args(m)
	rMsg := m
	var err error
	if isReply {
		rMsg, err = getRepliedMessage(c, m)
		if err != nil {
			_, err = m.ReplyText(c, "Invalid reply message.", nil)
			return err
		}
	}
	url := getUrl(rMsg)
	if isReply && args == "" && url == "" {
		args = rMsg.Text()
	}

	input := coalesce(url, args)

	if strings.HasPrefix(input, "tgpl_") {
		playlist, err := db.Instance.GetPlaylist(input)
		if err != nil {
			c.Logger.Warnf("load playlist %s in chat %d: %v", input, chatID, err)
			_, err = m.ReplyText(c, "❌ Playlist not found.", nil)
			if err != nil {
				c.Logger.Warnf("send playlist load failure in chat %d: %v", chatID, err)
			}
			return err
		}

		tracks := db.ConvertSongsToTracks(playlist.Songs)
		if len(tracks) == 0 {
			_, err = m.ReplyText(c, "❌ Playlist is empty.", nil)
			return err
		}

		updater, err := m.ReplyText(c, "🔍 Searching playlist...", nil)
		if err != nil {
			c.Logger.Warnf("send playlist search progress in chat %d: %v", chatID, err)
			return td.EndGroups
		}

		return handleMultipleTracks(c, m, updater, tracks, chatID, isVideo, force)
	}

	if match := utils.TelegramMessageRegex.FindStringSubmatch(input); match != nil {
		rMsg, err = utils.GetMessage(c, input)
		if err != nil {
			c.Logger.Warn("failed to parse message", "error", err.Error())
			_, err = m.ReplyText(c, "Invalid Telegram link.", nil)
			return err
		}
	}

	if isValid := isValidMedia(rMsg); isValid {
		isReply = true
	}

	if url == "" && args == "" && (!isReply || !isValidMedia(rMsg)) {
		if !isVideo && !force && !isReply && !m.IsPrivate() && m.SenderID() > 0 && !cache.ChatCache.IsActive(chatID) {
			if _, adminErr := cache.GetUserAdmin(c, chatID, m.SenderID(), false); adminErr == nil {
				tracks := getProfileTracks(c, m.SenderID())
				if len(tracks) > 0 && !cache.ChatCache.IsActive(chatID) {
					updater, sendErr := m.ReplyText(c, "🔍 Loading profile songs...", nil)
					if sendErr != nil {
						return sendErr
					}
					return handleMultipleTracks(c, m, updater, tracks, chatID, false, false)
				}
			}
		}
		_, _ = m.ReplyText(c, "<b>Usage:</b>\n/play [song or URL]\n\n<b>Supported Platforms:</b>\n- YouTube\n- Spotify\n- JioSaavn\n- Apple Music", &td.SendMessageOpts{ReplyMarkup: utils.SupportKeyboard(), ParseMode: "HTML"})
		return td.EndGroups
	}

	updater, err := m.ReplyText(c, "🔍 Searching and downloading...", nil)
	if err != nil {
		c.Logger.Warnf("send playback progress in chat %d: %v", chatID, err)
		return td.EndGroups
	}

	if isReply && isValidMedia(rMsg) {
		return handleMedia(c, m, updater, rMsg, chatID, isVideo, force)
	}

	wrapper := downloader.NewDlWrapper(input)
	if url != "" {
		if !wrapper.IsValid() {
			_, _ = updater.EditText(c, "Invalid URL or unsupported platform.\n\n<b>Supported Platforms:</b>\n- YouTube\n- Spotify\n- JioSaavn\n- Apple Music", &td.EditMessageTextOpts{ReplyMarkup: utils.SupportKeyboard(), ParseMode: "HTML"})
			return td.EndGroups
		}

		trackInfo, err := wrapper.GetInfo()
		if err != nil {
			c.Logger.Errorf("load track information in chat %d: %v", chatID, err)
			if _, editErr := updater.EditText(c, "Unable to load this track. It may be unavailable or restricted. Please try another.", nil); editErr != nil {
				return fmt.Errorf("edit track information error in chat %d: %w", chatID, editErr)
			}
			return td.EndGroups
		}

		if trackInfo.Results == nil || len(trackInfo.Results) == 0 {
			if _, editErr := updater.EditText(c, "No tracks found.", nil); editErr != nil {
				return fmt.Errorf("edit empty track result in chat %d: %w", chatID, editErr)
			}
			return td.EndGroups
		}

		return handleUrl(c, m, updater, trackInfo, chatID, isVideo, force)
	}

	return handleTextSearch(c, m, updater, wrapper, chatID, isVideo, force)
}

// handleMedia handles playing media from a message.
func handleMedia(c *td.Client, m *td.Message, updater *td.Message, dlMsg *td.Message, chatId int64, isVideo bool, force bool) error {
	file, fileName := getFile(dlMsg)
	if file == nil {
		_, err := updater.EditText(c, "No valid media found in the message.", nil)
		if err != nil {
			return fmt.Errorf("edit invalid media message in chat %d: %w", chatId, err)
		}
		return nil
	}

	if file.Size > config.MaxFileSize {
		_, err := updater.EditText(c, fmt.Sprintf("File too large. Max size: %d MB.", config.MaxFileSize/(1024*1024)), nil)
		if err != nil {
			return fmt.Errorf("edit oversized media message in chat %d: %w", chatId, err)
		}
		return nil
	}

	fileId := dlMsg.RemoteFileID()
	if _track := cache.ChatCache.GetTrackIfExists(chatId, fileId); _track != nil {
		_, err := updater.EditText(c, "Track already in queue or playing.", nil)
		if err != nil {
			return fmt.Errorf("edit duplicate media message in chat %d: %w", chatId, err)
		}
		return nil
	}

	dur := dlMsg.RemoteDuration()
	var linkURL string
	link, err := dlMsg.GetLink(c)
	if err != nil {
		c.Logger.Warn("Failed to get file link", "error", err)
	} else if link != nil {
		linkURL = link.Link
	}

	saveCache := utils.PlayerCache{
		URL: linkURL, Name: fileName, User: firstName(c, m), TrackID: fileId,
		Duration: dur, IsVideo: isVideo, Platform: utils.Telegram,
	}
	if audio, ok := dlMsg.Content.(*td.MessageAudio); ok && audio.Audio != nil {
		saveCache.Channel = audio.Audio.Performer
	}

	if queued, err := enqueueTrack(c, updater, chatId, &saveCache, force); queued {
		return err
	}

	filePath, err := downloader.DlCachedTrack(&saveCache, c, dlMsg)
	if err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		c.Logger.Errorf("download Telegram media in chat %d: %v", chatId, err)
		_, editErr := updater.EditText(c, "Unable to download this Telegram media. Please try again.", nil)
		if editErr != nil {
			return fmt.Errorf("edit media download error in chat %d: %w", chatId, editErr)
		}
		return nil
	}

	if dur == 0 {
		dur = utils.GetMediaDuration(filePath)
		saveCache.Duration = dur
	}

	saveCache.FilePath = filePath

	if err = calls.Calls.PlayMedia(c, chatId, saveCache.FilePath, saveCache.IsVideo); err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		c.Logger.Errorf("start media playback in chat %d: %v", chatId, err)
		_, editErr := updater.EditText(c, err.Error(), &td.EditMessageTextOpts{ParseMode: "HTML", DisableWebPagePreview: true})
		if editErr != nil {
			return fmt.Errorf("edit media playback error in chat %d: %w", chatId, editErr)
		}
		return nil
	}

	return sendStartedStreaming(c, updater, &saveCache)
}

// handleTextSearch handles a text search for a song.
func handleTextSearch(c *td.Client, m *td.Message, updater *td.Message, wrapper *downloader.DlWrapper, chatId int64, isVideo bool, force bool) error {
	searchResult, err := wrapper.Search()
	if err != nil {
		c.Logger.Errorf("search tracks in chat %d: %v", chatId, err)
		_, editErr := updater.EditText(c, "Search failed. Please try another query.", nil)
		if editErr != nil {
			return fmt.Errorf("edit track search error in chat %d: %w", chatId, editErr)
		}
		return nil
	}

	if searchResult.Results == nil || len(searchResult.Results) == 0 {
		_, err = updater.EditText(c, "😕 No results found. Try a different query.", nil)
		if err != nil {
			return fmt.Errorf("edit empty search result in chat %d: %w", chatId, err)
		}
		return nil
	}

	song := searchResult.Results[0]
	if _track := cache.ChatCache.GetTrackIfExists(chatId, song.Id); _track != nil {
		_, err := updater.EditText(c, "Track already in queue or playing.", nil)
		if err != nil {
			return fmt.Errorf("edit duplicate search track in chat %d: %w", chatId, err)
		}
		return nil
	}

	return handleSingleTrack(c, m, updater, song, "", chatId, isVideo, force)
}

// handleUrl handles a URL search for a song.
func handleUrl(c *td.Client, m *td.Message, updater *td.Message, trackInfo *utils.PlatformTracks, chatId int64, isVideo bool, force bool) error {
	if len(trackInfo.Results) == 1 {
		track := trackInfo.Results[0]
		if _track := cache.ChatCache.GetTrackIfExists(chatId, track.Id); _track != nil {
			_, err := updater.EditText(c, "Track already in queue or playing.", nil)
			if err != nil {
				return fmt.Errorf("edit duplicate URL track in chat %d: %w", chatId, err)
			}
			return nil
		}
		return handleSingleTrack(c, m, updater, track, "", chatId, isVideo, force)
	}

	return handleMultipleTracks(c, m, updater, trackInfo.Results, chatId, isVideo, force)
}

// handleSingleTrack handles a single track.
func handleSingleTrack(c *td.Client, m *td.Message, updater *td.Message, song utils.GetUrlTrack, filePath string, chatId int64, isVideo bool, force bool) error {
	if song.Duration > config.SongDurationLimit {
		_, err := updater.EditText(c, fmt.Sprintf("Sorry, song exceeds max duration of %d minutes.", config.SongDurationLimit/60), nil)
		if err != nil {
			return fmt.Errorf("edit long track message in chat %d: %w", chatId, err)
		}
		return nil
	}

	saveCache := utils.PlayerCache{
		URL: song.Url, Name: song.Title, User: firstName(c, m), FilePath: filePath,
		Thumbnail: song.Thumbnail, TrackID: song.Id, Duration: song.Duration, Channel: song.Channel, Views: song.Views,
		IsVideo: isVideo, Platform: song.Platform,
	}

	if queued, err := enqueueTrack(c, updater, chatId, &saveCache, force); queued {
		return err
	}

	if saveCache.FilePath == "" {
		dlResult, err := downloader.DlCachedTrack(&saveCache, c, nil)
		if err != nil {
			cache.ChatCache.RemoveCurrentSong(chatId)
			c.Logger.Errorf("download track in chat %d: %v", chatId, err)
			_, editErr := updater.EditText(c, "Unable to download this track. It may be restricted or unavailable. Please try another.", nil)
			if editErr != nil {
				return fmt.Errorf("edit track download error in chat %d: %w", chatId, editErr)
			}
			return nil
		}

		saveCache.FilePath = dlResult
	}

	if err := calls.Calls.PlayMedia(c, chatId, saveCache.FilePath, saveCache.IsVideo); err != nil {
		cache.ChatCache.RemoveCurrentSong(chatId)
		c.Logger.Errorf("start track playback in chat %d: %v", chatId, err)
		_, editErr := updater.EditText(c, err.Error(), &td.EditMessageTextOpts{ParseMode: "HTML", DisableWebPagePreview: true})
		if editErr != nil {
			return fmt.Errorf("edit track playback error in chat %d: %w", chatId, editErr)
		}
		return nil
	}

	if err := sendStartedStreaming(c, updater, &saveCache); err != nil {
		return err
	}

	return nil
}

// handleMultipleTracks handles multiple tracks.
func handleMultipleTracks(c *td.Client, m *td.Message, updater *td.Message, tracks []utils.GetUrlTrack, chatId int64, isVideo bool, force bool) error {
	if len(tracks) == 0 {
		_, err := updater.EditText(c, "No tracks found.", nil)
		if err != nil {
			return fmt.Errorf("edit empty queue message in chat %d: %w", chatId, err)
		}
		return nil
	}

	queueHeader := "<u><b>Added to Queue:</b></u>\n<blockquote expandable>\n"
	var tracksToAdd []*utils.PlayerCache
	var skippedTracks []string

	for _, track := range tracks {
		if track.Duration > config.SongDurationLimit {
			skippedTracks = append(skippedTracks, track.Title)
			continue
		}

		saveCache := &utils.PlayerCache{
			Name: track.Title, TrackID: track.Id, Duration: track.Duration,
			Thumbnail: track.Thumbnail, User: firstName(c, m), Platform: track.Platform,
			IsVideo: isVideo, URL: track.Url, Channel: track.Channel, Views: track.Views,
		}
		tracksToAdd = append(tracksToAdd, saveCache)
	}

	if len(tracksToAdd) == 0 {
		if len(skippedTracks) > 0 {
			_, err := updater.EditText(c, fmt.Sprintf("All tracks were skipped (max duration %d min).", config.SongDurationLimit/60), nil)
			if err != nil {
				return fmt.Errorf("edit skipped tracks message in chat %d: %w", chatId, err)
			}
			return nil
		}
		_, err := updater.EditText(c, "No valid tracks found.", nil)
		if err != nil {
			return fmt.Errorf("edit invalid tracks message in chat %d: %w", chatId, err)
		}
		return nil
	}

	var qLenAfter int
	var startLen int

	if force {
		qLenAfter = 0
		for _, t := range slices.Backward(tracksToAdd) {
			qLenAfter = cache.ChatCache.AddSongToFront(chatId, t)
		}
		startLen = qLenAfter - len(tracksToAdd)
		if startLen > 0 {
			_ = calls.Calls.PlayNext(c, chatId)
			_ = c.DeleteMessages(chatId, []int64{updater.Id}, &td.DeleteMessagesOpts{Revoke: true})
			return nil
		}
	} else {
		qLenAfter = cache.ChatCache.AddSongs(chatId, tracksToAdd)
		startLen = qLenAfter - len(tracksToAdd)
	}

	if startLen == 0 {
		return calls.Calls.PlayTrackWithMessage(c, updater, chatId, tracksToAdd[0])
	}

	var sb strings.Builder
	sb.WriteString(queueHeader)

	var totalDuration int32
	for i, track := range tracksToAdd {
		currentQLen := startLen + i + 1
		escTrackName := html.EscapeString(track.Name)
		fmt.Fprintf(&sb, "<b>%d.</b> %s\n└ Duration: %s\n",
			currentQLen, escTrackName, utils.SecToMin(track.Duration))
		totalDuration += track.Duration
	}

	sb.WriteString("</blockquote>")
	escRequester := html.EscapeString(firstName(c, m))
	queueSummary := fmt.Sprintf(
		"\n<b>Queue Total:</b> %d\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
		qLenAfter, utils.SecToMin(totalDuration), escRequester,
	)

	sb.WriteString(queueSummary)
	if len(skippedTracks) > 0 {
		fmt.Fprintf(&sb, "\n\n<b>Skipped %d tracks</b> (exceeded duration limit).", len(skippedTracks))
	}

	fullMessage := sb.String()

	if len(fullMessage) > 4096 {
		fullMessage = queueSummary
	}

	_, err := updater.EditText(c, fullMessage, &td.EditMessageTextOpts{
		ParseMode:             "HTML",
		ReplyMarkup:           utils.QueueMarkup(tracksToAdd[0].TrackID),
		DisableWebPagePreview: true,
	})
	if err != nil {
		if strings.Contains(err.Error(), "MESSAGE_NOT_MODIFIED") {
			return nil
		}
		return fmt.Errorf("edit queue summary in chat %d: %w", chatId, err)
	}
	return nil
}

func enqueueTrack(c *td.Client, updater *td.Message, chatId int64, saveCache *utils.PlayerCache, force bool) (bool, error) {
	var qLen int
	if force {
		qLen = cache.ChatCache.AddSongToFront(chatId, saveCache)
	} else {
		qLen = cache.ChatCache.AddSong(chatId, saveCache)
	}

	if qLen > 1 {
		if force {
			_ = calls.Calls.PlayNext(c, chatId)
			_ = c.DeleteMessages(chatId, []int64{updater.Id}, &td.DeleteMessagesOpts{Revoke: true})
			return true, nil
		}
		escURL := html.EscapeString(saveCache.URL)
		escName := html.EscapeString(saveCache.Name)
		escUser := html.EscapeString(saveCache.User)
		queueInfo := fmt.Sprintf(
			"<u><b>Added to queue: %d</b></u>\n\n<b>Title:</b> <a href='%s'>%s</a>\n\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
			qLen, escURL, escName, utils.SecToMin(saveCache.Duration), escUser,
		)

		_, err := updater.EditText(c, queueInfo, &td.EditMessageTextOpts{
			ReplyMarkup:           utils.QueueMarkup(saveCache.TrackID),
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
		})
		if err != nil && !strings.Contains(err.Error(), "MESSAGE_NOT_MODIFIED") {
			return true, fmt.Errorf("edit queued track in chat %d: %w", chatId, err)
		}
		return true, nil
	}

	return false, nil
}

func sendStartedStreaming(c *td.Client, updater *td.Message, saveCache *utils.PlayerCache) error {
	escURL := html.EscapeString(saveCache.URL)
	escName := html.EscapeString(saveCache.Name)
	escUser := html.EscapeString(saveCache.User)

	nowPlaying := fmt.Sprintf(
		"<u><b>| Started streaming</b></u>\n\n<b>Title:</b> <a href='%s'>%s</a>\n\n<b>Duration:</b> %s min\n<b>Requested by:</b> %s",
		escURL, escName, utils.SecToMin(saveCache.Duration), escUser,
	)

	_, err := updater.EditText(c, nowPlaying, &td.EditMessageTextOpts{
		ParseMode:             "HTML",
		ReplyMarkup:           utils.ControlButtons("play"),
		DisableWebPagePreview: true,
	})
	if err != nil {
		if strings.Contains(err.Error(), "MESSAGE_NOT_MODIFIED") {
			return nil
		}
		return fmt.Errorf("edit started track in chat %d: %w", updater.ChatId, err)
	}
	return nil
}
