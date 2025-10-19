package handlers

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AshokShau/TgMusicBot/pkg/config"
	"github.com/AshokShau/TgMusicBot/pkg/core"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/core/dl"
	"github.com/AshokShau/TgMusicBot/pkg/lang"
	"github.com/AshokShau/TgMusicBot/pkg/vc"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

// statusUpdater is a wrapper around telegram.NewMessage to prevent flood waits.
type statusUpdater struct {
	*telegram.NewMessage
	mu          sync.Mutex
	lastMessage string
	lastSent    time.Time
}

// Edit edits the message, but only if the content has changed, and it has been more than 500ms since the last edit.
func (su *statusUpdater) Edit(text string, opts ...telegram.SendOptions) (*telegram.NewMessage, error) {
	su.mu.Lock()
	defer su.mu.Unlock()

	if text == su.lastMessage {
		return su.NewMessage, nil
	}

	if time.Since(su.lastSent) < 500*time.Millisecond {
		time.Sleep(500*time.Millisecond - time.Since(su.lastSent))
	}

	msg, err := su.NewMessage.Edit(text, opts...)
	if err == nil {
		su.lastMessage = text
		su.lastSent = time.Now()
	}
	return msg, err
}

// playHandler handles the /play command.
func playHandler(m *telegram.NewMessage) error {
	return handlePlay(m, false)
}

// vPlayHandler handles the /vplay command.
func vPlayHandler(m *telegram.NewMessage) error {
	return handlePlay(m, true)
}

// handlePlay is the main handler for /play and /vplay commands.
func handlePlay(m *telegram.NewMessage, isVideo bool) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if queue := cache.ChatCache.GetQueue(chatID); len(queue) > 10 {
		_, err := m.Reply(lang.GetString(langCode, "play_queue_full"))
		return err
	}

	isReply := m.IsReply()
	url := getUrl(m, isReply)
	args := m.Args()
	rMsg := m
	var err error

	parseTelegramURL := func(input string) (string, int, bool) {
		re := regexp.MustCompile(`^https://t\.me/([a-zA-Z0-9_]{4,})/(\d+)$`)
		matches := re.FindStringSubmatch(input)
		if matches == nil {
			return "", 0, false
		}
		id, err := strconv.Atoi(matches[2])
		if err != nil {
			return "", 0, false
		}
		return matches[1], id, true
	}

	input := coalesce(url, args)
	if username, msgID, ok := parseTelegramURL(input); ok {
		rMsg, err = m.Client.GetMessageByID(username, int32(msgID))
		if err != nil {
			_, err = m.Reply(lang.GetString(langCode, "play_invalid_tg_link"))
			return err
		}
	} else if isReply {
		rMsg, err = m.GetReplyMessage()
		if err != nil {
			_, err = m.Reply(lang.GetString(langCode, "play_invalid_reply"))
			return err
		}
	}

	if isValid := isValidMedia(rMsg); isValid {
		isReply = true
	}

	if url == "" && args == "" && (!isReply || !isValidMedia(rMsg)) {
		_, err := m.Reply(lang.GetString(langCode, "play_usage"), telegram.SendOptions{ReplyMarkup: core.SupportKeyboard()})
		return err
	}

	statusMsg, err := m.Reply(lang.GetString(langCode, "play_searching"))
	if err != nil {
		gologging.WarnF("failed to send message: %v", err)
		return err
	}

	updater := &statusUpdater{NewMessage: statusMsg, lastMessage: lang.GetString(langCode, "play_searching"), lastSent: time.Now()}

	if isReply && isValidMedia(rMsg) {
		return handleMedia(m, updater, rMsg, chatID, isVideo, langCode)
	}

	wrapper := dl.NewDownloaderWrapper(input)
	if url != "" {
		if !wrapper.IsValid() {
			_, err = updater.Edit(lang.GetString(langCode, "play_invalid_url"), telegram.SendOptions{ReplyMarkup: core.SupportKeyboard()})
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		trackInfo, err := wrapper.GetInfo(ctx)
		if err != nil {
			_, err = updater.Edit(fmt.Sprintf(lang.GetString(langCode, "play_fetch_error"), err.Error()))
			return err
		}
		if trackInfo.Results == nil {
			_, err = updater.Edit(lang.GetString(langCode, "play_no_tracks_found"))
			return err
		}
		return handleUrl(m, updater, trackInfo, chatID, isVideo, langCode)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()
	return handleTextSearch(m, updater, wrapper, chatID, isVideo, ctx2, langCode)
}

// handleMedia handles playing media from a message.
func handleMedia(m *telegram.NewMessage, updater *statusUpdater, dlMsg *telegram.NewMessage, chatId int64, isVideo bool, langCode string) error {
	if dlMsg.File.Size > config.Conf.MaxFileSize {
		_, err := updater.Edit(fmt.Sprintf(lang.GetString(langCode, "play_file_too_large"), config.Conf.MaxFileSize/(1024*1024)))
		if err != nil {
			gologging.WarnF("[play.go - handleMedia] Edit message failed: %v", err)
		}
		return nil
	}

	fileName := dlMsg.File.Name
	fileId := dlMsg.File.FileID

	if _track := cache.ChatCache.GetTrackIfExists(chatId, fileId); _track != nil {
		_, err := updater.Edit(lang.GetString(langCode, "play_track_already_in_queue"))
		if err != nil {
			gologging.InfoF("[play.go - handleMedia] Edit message failed: %v", err)
		}
		return nil
	}

	dur := cache.GetFileDur(dlMsg)
	if cache.ChatCache.IsActive(chatId) {
		saveCache := cache.CachedTrack{
			URL: dlMsg.Link(), Name: fileName, User: m.Sender.FirstName, TrackID: fileId,
			Duration: dur, IsVideo: isVideo, Platform: cache.Telegram,
		}
		queue := cache.ChatCache.GetQueue(chatId)
		cache.ChatCache.AddSong(chatId, &saveCache)
		queueInfo := fmt.Sprintf(
			lang.GetString(langCode, "play_added_to_queue"),
			len(queue), saveCache.URL, saveCache.Name, cache.SecToMin(saveCache.Duration), saveCache.User,
		)
		_, err := updater.Edit(queueInfo, telegram.SendOptions{ReplyMarkup: core.ControlButtons("play")})
		if err != nil {
			gologging.WarnF("[play.go - handleMedia] Edit message failed: %v", err)
		}
		return nil
	}

	filePath, err := dlMsg.Download(&telegram.DownloadOptions{FileName: filepath.Join(config.Conf.DownloadsDir, fileName)})
	if err != nil {
		_, err = updater.Edit(fmt.Sprintf(lang.GetString(langCode, "play_download_failed"), err.Error()))
		return err
	}

	if dur == 0 {
		dur = cache.GetFileDuration(filePath)
	}

	time.Sleep(200 * time.Millisecond)
	track := cache.MusicTrack{
		Name: fileName, Duration: dur, URL: dlMsg.Link(), ID: fileId, Platform: cache.Telegram,
	}
	return handleSingleTrack(m, updater, track, filePath, chatId, isVideo, langCode)
}

// handleTextSearch handles a text search for a song.
func handleTextSearch(m *telegram.NewMessage, updater *statusUpdater, wrapper *dl.DownloaderWrapper, chatId int64, isVideo bool, ctx context.Context, langCode string) error {
	searchResult, err := wrapper.Search(ctx)
	if err != nil {
		_, err = updater.Edit(fmt.Sprintf(lang.GetString(langCode, "play_search_failed"), err.Error()))
		return err
	}

	if searchResult.Results == nil || len(searchResult.Results) == 0 {
		_, err = updater.Edit(lang.GetString(langCode, "play_no_results"))
		return err
	}

	song := searchResult.Results[0]
	if _track := cache.ChatCache.GetTrackIfExists(chatId, song.ID); _track != nil {
		_, err := updater.Edit(lang.GetString(langCode, "play_track_already_in_queue"))
		return err
	}

	return handleSingleTrack(m, updater, song, "", chatId, isVideo, langCode)
}

// handleUrl handles a URL search for a song.
func handleUrl(m *telegram.NewMessage, updater *statusUpdater, trackInfo cache.PlatformTracks, chatId int64, isVideo bool, langCode string) error {
	if len(trackInfo.Results) == 1 {
		track := trackInfo.Results[0]
		if _track := cache.ChatCache.GetTrackIfExists(chatId, track.ID); _track != nil {
			_, err := updater.Edit(lang.GetString(langCode, "play_track_already_in_queue"))
			return err
		}
		return handleSingleTrack(m, updater, track, "", chatId, isVideo, langCode)
	}
	return handleMultipleTracks(m, updater, trackInfo.Results, chatId, isVideo, langCode)
}

// handleSingleTrack handles a single track.
func handleSingleTrack(m *telegram.NewMessage, updater *statusUpdater, song cache.MusicTrack, filePath string, chatId int64, isVideo bool, langCode string) error {
	saveCache := cache.CachedTrack{
		URL: song.URL, Name: song.Name, User: m.Sender.FirstName, FilePath: filePath,
		Thumbnail: song.Cover, TrackID: song.ID, Duration: song.Duration,
		IsVideo: isVideo, Platform: song.Platform,
	}

	if cache.ChatCache.IsActive(chatId) {
		queue := cache.ChatCache.GetQueue(chatId)
		cache.ChatCache.AddSong(chatId, &saveCache)
		queueInfo := fmt.Sprintf(
			lang.GetString(langCode, "play_added_to_queue"),
			len(queue), saveCache.URL, saveCache.Name, cache.SecToMin(saveCache.Duration), saveCache.User,
		)
		_, err := updater.Edit(queueInfo, telegram.SendOptions{ReplyMarkup: core.ControlButtons("play")})
		if err != nil {
			gologging.WarnF("[play.go - handleSingleTrack] Edit message failed: %v", err)
		}
		return nil
	}

	if saveCache.FilePath == "" {
		_, err := updater.Edit(fmt.Sprintf(lang.GetString(langCode, "downloading"), song.Name))
		if err != nil {
			gologging.WarnF("[play.go - handleSingleTrack] Edit message failed: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		dlResult, trackInfo, err := vc.DownloadSong(ctx, &saveCache, m.Client)
		if err != nil {
			_, err = updater.Edit(fmt.Sprintf(lang.GetString(langCode, "play_song_download_failed"), err.Error()))
			return err
		}

		saveCache.FilePath = dlResult
		if trackInfo != nil {
			saveCache.Lyrics = trackInfo.Lyrics
			if song.Duration == 0 {
				saveCache.Duration = trackInfo.Duration
			}
		}
	}

	cache.ChatCache.SetActive(chatId, true)
	cache.ChatCache.AddSong(chatId, &saveCache)

	if err := vc.Calls.PlayMedia(chatId, saveCache.FilePath, saveCache.IsVideo, ""); err != nil {
		_, err = updater.Edit(err.Error())
		return err
	}

	nowPlaying := fmt.Sprintf(
		lang.GetString(langCode, "play_now_playing"),
		saveCache.URL, saveCache.Name, cache.SecToMin(song.Duration), saveCache.User,
	)
	_, err := updater.Edit(nowPlaying, telegram.SendOptions{ReplyMarkup: core.ControlButtons("play")})
	if err != nil {
		gologging.WarnF("[play.go - handleSingleTrack] Edit message failed: %v", err)
	}
	return nil
}

// handleMultipleTracks handles multiple tracks.
func handleMultipleTracks(m *telegram.NewMessage, updater *statusUpdater, tracks []cache.MusicTrack, chatId int64, isVideo bool, langCode string) error {
	isActive := cache.ChatCache.IsActive(chatId)
	queue := cache.ChatCache.GetQueue(chatId)
	queueHeader := lang.GetString(langCode, "play_added_to_queue_header")
	var queueItems []string

	for i, track := range tracks {
		position := len(queue) + i
		saveCache := cache.CachedTrack{
			Name: track.Name, TrackID: track.ID, Duration: track.Duration,
			Thumbnail: track.Cover, User: m.Sender.FirstName, Platform: track.Platform,
			IsVideo: isVideo, URL: track.URL,
		}
		if !isActive && i == 0 {
			saveCache.Loop = 1
		}
		cache.ChatCache.AddSong(chatId, &saveCache)
		queueItems = append(queueItems, fmt.Sprintf(lang.GetString(langCode, "play_queue_item"), position, track.Name, cache.SecToMin(track.Duration)))
	}

	totalDuration := 0
	for _, t := range tracks {
		totalDuration += t.Duration
	}

	queueSummary := fmt.Sprintf(
		lang.GetString(langCode, "play_queue_summary"),
		len(cache.ChatCache.GetQueue(chatId)), cache.SecToMin(totalDuration), m.Sender.FirstName,
	)
	fullMessage := queueHeader + strings.Join(queueItems, "\n") + queueSummary
	if len(fullMessage) > 4096 {
		fullMessage = queueSummary
	}

	if !isActive {
		_ = vc.Calls.PlayNext(chatId)
	}

	_, err := updater.Edit(fullMessage, telegram.SendOptions{ReplyMarkup: core.ControlButtons("play")})
	if err != nil {
		gologging.WarnF("[play.go - handleMultipleTracks] Edit message failed: %v", err)
	}
	return nil
}
