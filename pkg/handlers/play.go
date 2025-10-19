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
	"github.com/AshokShau/TgMusicBot/pkg/core/dl"
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
	if queue := cache.ChatCache.GetQueue(chatID); len(queue) > 10 {
		_, err := m.Reply("⚠️ The queue is full (10 tracks max). Use /end to clear it.")
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
			_, err = m.Reply("❌ The provided Telegram link is invalid.")
			return err
		}
	} else if isReply {
		rMsg, err = m.GetReplyMessage()
		if err != nil {
			_, err = m.Reply("❌ The replied-to message is not valid.")
			return err
		}
	}

	if isValid := isValidMedia(rMsg); isValid {
		isReply = true
	}

	if url == "" && args == "" && (!isReply || !isValidMedia(rMsg)) {
		_, err := m.Reply("🎵 <b>Usage:</b>\n/play [song name or URL]\n\n<b>Supported Platforms:</b>\n- YouTube\n- Spotify\n- JioSaavn\n- Apple Music", telegram.SendOptions{ReplyMarkup: core.SupportKeyboard()})
		return err
	}

	statusMsg, err := m.Reply("🔍 Searching...")
	if err != nil {
		gologging.WarnF("failed to send message: %v", err)
		return err
	}

	updater := &statusUpdater{NewMessage: statusMsg, lastMessage: "🔍 Searching...", lastSent: time.Now()}

	if isReply && isValidMedia(rMsg) {
		return handleMedia(m, updater, rMsg, chatID, isVideo)
	}

	wrapper := dl.NewDownloaderWrapper(input)
	if url != "" {
		if !wrapper.IsValid() {
			_, err = updater.Edit("❌ Invalid URL or unsupported platform.\n\n<b>Supported Platforms:</b>\n- YouTube\n- Spotify\n- JioSaavn\n- Apple Music", telegram.SendOptions{ReplyMarkup: core.SupportKeyboard()})
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		trackInfo, err := wrapper.GetInfo(ctx)
		if err != nil {
			_, err = updater.Edit("❌ Error fetching track information: " + err.Error())
			return err
		}
		if trackInfo.Results == nil {
			_, err = updater.Edit("❌ No tracks were found for the provided source.")
			return err
		}
		return handleUrl(m, updater, trackInfo, chatID, isVideo)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return handleTextSearch(m, updater, wrapper, chatID, isVideo, ctx)
}

// handleMedia handles playing media from a message.
func handleMedia(m *telegram.NewMessage, updater *statusUpdater, dlMsg *telegram.NewMessage, chatId int64, isVideo bool) error {
	if dlMsg.File.Size > config.Conf.MaxFileSize {
		_, err := updater.Edit(fmt.Sprintf("❌ File size is too large. The maximum allowed size is %d MB.", config.Conf.MaxFileSize/(1024*1024)))
		if err != nil {
			gologging.WarnF("[play.go - handleMedia] Edit message failed: %v", err)
		}
		return nil
	}

	fileName := dlMsg.File.Name
	fileId := dlMsg.File.FileID

	if _track := cache.ChatCache.GetTrackIfExists(chatId, fileId); _track != nil {
		_, err := updater.Edit("✅ This track is already in the queue or currently playing.")
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
			"<b>🎧 Added to Queue (#%d)</b>\n\n"+
				"▫ <b>Track:</b> <a href='%s'>%s</a>\n"+
				"▫ <b>Duration:</b> %s\n"+
				"▫ <b>Requested by:</b> %s",
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
		_, err = updater.Edit("❌ Failed to download the media: " + err.Error())
		return err
	}

	if dur == 0 {
		dur = cache.GetFileDuration(filePath)
	}

	time.Sleep(200 * time.Millisecond)
	track := cache.MusicTrack{
		Name: fileName, Duration: dur, URL: dlMsg.Link(), ID: fileId, Platform: cache.Telegram,
	}
	return handleSingleTrack(m, updater, track, filePath, chatId, isVideo)
}

// handleTextSearch handles a text search for a song.
func handleTextSearch(m *telegram.NewMessage, updater *statusUpdater, wrapper *dl.DownloaderWrapper, chatId int64, isVideo bool, ctx context.Context) error {
	searchResult, err := wrapper.Search(ctx)
	if err != nil {
		_, err = updater.Edit("❌ Search failed: " + err.Error())
		return err
	}

	if searchResult.Results == nil || len(searchResult.Results) == 0 {
		_, err = updater.Edit("😕 No results found. Please try a different search query.")
		return err
	}

	song := searchResult.Results[0]
	if _track := cache.ChatCache.GetTrackIfExists(chatId, song.ID); _track != nil {
		_, err := updater.Edit("✅ This track is already in the queue or currently playing.")
		return err
	}

	return handleSingleTrack(m, updater, song, "", chatId, isVideo)
}

// handleUrl handles a URL search for a song.
func handleUrl(m *telegram.NewMessage, updater *statusUpdater, trackInfo cache.PlatformTracks, chatId int64, isVideo bool) error {
	if len(trackInfo.Results) == 1 {
		track := trackInfo.Results[0]
		if _track := cache.ChatCache.GetTrackIfExists(chatId, track.ID); _track != nil {
			_, err := updater.Edit("✅ This track is already in the queue or currently playing.")
			return err
		}
		return handleSingleTrack(m, updater, track, "", chatId, isVideo)
	}
	return handleMultipleTracks(m, updater, trackInfo.Results, chatId, isVideo)
}

// handleSingleTrack handles a single track.
func handleSingleTrack(m *telegram.NewMessage, updater *statusUpdater, song cache.MusicTrack, filePath string, chatId int64, isVideo bool) error {
	saveCache := cache.CachedTrack{
		URL: song.URL, Name: song.Name, User: m.Sender.FirstName, FilePath: filePath,
		Thumbnail: song.Cover, TrackID: song.ID, Duration: song.Duration,
		IsVideo: isVideo, Platform: song.Platform,
	}

	if cache.ChatCache.IsActive(chatId) {
		queue := cache.ChatCache.GetQueue(chatId)
		cache.ChatCache.AddSong(chatId, &saveCache)
		queueInfo := fmt.Sprintf(
			"<b>🎧 Added to Queue (#%d)</b>\n\n"+
				"▫ <b>Track:</b> <a href='%s'>%s</a>\n"+
				"▫ <b>Duration:</b> %s\n"+
				"▫ <b>Requested by:</b> %s",
			len(queue), saveCache.URL, saveCache.Name, cache.SecToMin(saveCache.Duration), saveCache.User,
		)
		_, err := updater.Edit(queueInfo, telegram.SendOptions{ReplyMarkup: core.ControlButtons("play")})
		if err != nil {
			gologging.WarnF("[play.go - handleSingleTrack] Edit message failed: %v", err)
		}
		return nil
	}

	if saveCache.FilePath == "" {
		_, err := updater.Edit("⬇️ Downloading...")
		if err != nil {
			gologging.WarnF("[play.go - handleSingleTrack] Edit message failed: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		dlResult, trackInfo, err := vc.DownloadSong(ctx, &saveCache, m.Client)
		if err != nil {
			_, err = updater.Edit("❌ Failed to download the song: " + err.Error())
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
		"🎵 <b>Now Playing:</b>\n\n"+
			"▫ <b>Track:</b> <a href='%s'>%s</a>\n"+
			"▫ <b>Duration:</b> %s\n"+
			"▫ <b>Requested by:</b> %s",
		saveCache.URL, saveCache.Name, cache.SecToMin(song.Duration), saveCache.User,
	)
	_, err := updater.Edit(nowPlaying, telegram.SendOptions{ReplyMarkup: core.ControlButtons("play")})
	if err != nil {
		gologging.WarnF("[play.go - handleSingleTrack] Edit message failed: %v", err)
	}
	return nil
}

// handleMultipleTracks handles multiple tracks.
func handleMultipleTracks(m *telegram.NewMessage, updater *statusUpdater, tracks []cache.MusicTrack, chatId int64, isVideo bool) error {
	isActive := cache.ChatCache.IsActive(chatId)
	queue := cache.ChatCache.GetQueue(chatId)
	queueHeader := "<b>📥 Added to Queue:</b>\n<blockquote expandable>\n"
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
		queueItems = append(queueItems, fmt.Sprintf("<b>%d.</b> %s\n└ Duration: %s", position, track.Name, cache.SecToMin(track.Duration)))
	}

	totalDuration := 0
	for _, t := range tracks {
		totalDuration += t.Duration
	}

	queueSummary := fmt.Sprintf(
		"</blockquote>\n<b>📋 Total in Queue:</b> %d\n<b>⏱ Total Duration:</b> %s\n<b>👤 Requested by:</b> %s",
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
