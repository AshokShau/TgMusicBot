package handlers

import (
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

var (
	streams   = make(map[int64]*activeStream)
	streamsMu sync.RWMutex

	rtmpRegex         = regexp.MustCompile(`^rtmps?:\/\/[A-Za-z0-9.-]+(?:\/[A-Za-z0-9._~:/?#@!$&'()*+,;=-]*)?$`)
	maxChunkSizeBytes = int64(1 << 20)
)

type activeStream struct {
	cmd       *exec.Cmd
	userID    int64
	fileName  string
	rtmpURL   string
	startTime time.Time
}

func isValidRTMP(url string) bool {
	return rtmpRegex.MatchString(url)
}

func streamExists(chatID int64) bool {
	streamsMu.RLock()
	defer streamsMu.RUnlock()
	_, exists := streams[chatID]
	return exists
}

func streamHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	ctx, cancel := db.Ctx()
	defer cancel()

	langCode := db.Instance.GetLang(ctx, chatID)

	rtmpURL, _ := db.Instance.GetRtmpUrl(ctx, chatID)
	if rtmpURL == "" || !isValidRTMP(rtmpURL) {
		_, _ = m.Reply(lang.GetString(langCode, "rtmp_missing"))
		return telegram.ErrEndGroup
	}

	if streamExists(chatID) {
		_, _ = m.Reply(lang.GetString(langCode, "stream_exists"))
		return telegram.ErrEndGroup
	}

	rMsg := resolveMediaInput(m)
	if rMsg == nil || !isValidMedia(rMsg) {
		_, _ = m.Reply(lang.GetString(langCode, "play_invalid"))
		return telegram.ErrEndGroup
	}

	statusMsg, _ := m.Reply("⏳ Preparing stream...")
	if err := startStream(rMsg, rtmpURL, m.SenderID(), chatID); err != nil {
		_, _ = statusMsg.Edit(fmt.Sprintf("❌ Failed: %v", err))
		return telegram.ErrEndGroup
	}

	_, _ = statusMsg.Edit(fmt.Sprintf(
		"🔴 <b>Live Stream Started</b>\n📂 <b>%s</b>\n\nUse <code>/stopstream</code> to stop.",
		rMsg.File.Name,
	))

	return telegram.ErrEndGroup
}

func resolveMediaInput(m *telegram.NewMessage) *telegram.NewMessage {
	if username, msgID, ok := parseTelegramURL(m.Args()); ok {
		msg, _ := m.Client.GetMessageByID(username, int32(msgID))
		return msg
	}

	if m.IsReply() {
		msg, _ := m.GetReplyMessage()
		return msg
	}

	return nil
}

func startStream(msg *telegram.NewMessage, rtmpURL string, userID, chatID int64) error {
	cmd := exec.Command("ffmpeg",
		"-re",
		"-stream_loop", "-1",
		"-i", "pipe:0",
		"-preset", "veryfast",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-b:v", "2000k",
		"-g", "30",
		"-c:a", "aac",
		"-b:a", "96k",
		"-f", "flv",
		rtmpURL,
	)

	ffIn, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("FFmpeg pipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("FFmpeg failed: %w", err)
	}

	streamsMu.Lock()
	streams[chatID] = &activeStream{
		cmd:       cmd,
		userID:    userID,
		fileName:  msg.File.Name,
		rtmpURL:   rtmpURL,
		startTime: time.Now(),
	}
	streamsMu.Unlock()

	go func() {
		defer ffIn.Close()

		for offset := int64(0); offset < msg.File.Size; offset += maxChunkSizeBytes {
			chunk, _, err := msg.Client.DownloadChunk(msg.Media(), int(offset), int(offset+maxChunkSizeBytes), int(maxChunkSizeBytes))
			if err != nil {
				logger.Warnf("Chunk read error chat%d: %v", chatID, err)
				return
			}

			if _, err := ffIn.Write(chunk); err != nil {
				logger.Warnf("Chunk write error chat%d: %v", chatID, err)
				return
			}
		}
	}()

	go func() {
		_ = cmd.Wait()
		stopStream(chatID)
		logger.Infof("Stream stopped for %d", chatID)
	}()

	return nil
}

func stopStream(chatID int64) {
	streamsMu.Lock()
	stream, exists := streams[chatID]
	if exists {
		delete(streams, chatID)
	}
	streamsMu.Unlock()

	if !exists {
		return
	}

	if stream.cmd.Process != nil {
		_ = stream.cmd.Process.Kill()
	}

	logger.Debugf("Stream manually stopped chat %d", chatID)
}

func stopStreamHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()

	if !streamExists(chatID) {
		_, _ = m.Reply("📭 No active stream in this chat.")
		return telegram.ErrEndGroup
	}

	stopStream(chatID)
	_, _ = m.Reply("🛑 Stream stopped.")
	return telegram.ErrEndGroup
}

func listStreamsHandler(m *telegram.NewMessage) error {
	streamsMu.RLock()
	defer streamsMu.RUnlock()

	if len(streams) == 0 {
		_, _ = m.Reply("📭 No active streams.")
		return nil
	}

	var b strings.Builder
	b.WriteString("<b>📡 Active Streams</b>\n\n")

	for chatID, s := range streams {
		b.WriteString(fmt.Sprintf("💬 Chat: <code>%d</code>\n📂 %s\n🕒 %s\n\n",
			chatID, s.fileName, time.Since(s.startTime).Round(time.Second)))
	}

	_, _ = m.Reply(b.String())
	return telegram.ErrEndGroup
}

func setRtmpHandler(m *telegram.NewMessage) error {
	ctx, cancel := db.Ctx()
	defer cancel()

	langCode := db.Instance.GetLang(ctx, m.ChannelID())

	if !m.IsPrivate() {
		_, _ = m.Reply(lang.GetString(langCode, "setrtmp_private_only"))
		return telegram.ErrEndGroup
	}

	args := strings.Fields(m.Args())
	if len(args) != 2 {
		_, _ = m.Reply(lang.GetString(langCode, "setrtmp_usage"))
		return telegram.ErrEndGroup
	}

	chatID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || !strings.HasPrefix(args[0], "-100") {
		_, _ = m.Reply(lang.GetString(langCode, "setrtmp_invalid_chat_id"))
		return telegram.ErrEndGroup
	}

	rtmpURL := args[1]
	if !isValidRTMP(rtmpURL) {
		_, _ = m.Reply(lang.GetString(langCode, "setrtmp_invalid_url"))
		return telegram.ErrEndGroup
	}

	client := m.Client

	if bot, err := cache.GetUserAdmin(client, chatID, client.Me().ID, false); err != nil || !bot.Rights.ManageCall {
		_, _ = m.Reply(lang.GetString(langCode, "setrtmp_bot_not_admin"))
		return telegram.ErrEndGroup
	}

	if user, err := cache.GetUserAdmin(client, chatID, m.SenderID(), false); err != nil || user.Rights == nil || !user.Rights.ManageCall {
		_, _ = m.Reply(lang.GetString(langCode, "setrtmp_user_not_admin"))
		return telegram.ErrEndGroup
	}

	if err := db.Instance.SetRtmpUrl(ctx, chatID, rtmpURL); err != nil {
		_, _ = m.Reply(fmt.Sprintf("❌ Error saving RTMP URL: %v", err))
		return telegram.ErrEndGroup
	}

	_, _ = m.Reply(fmt.Sprintf(lang.GetString(langCode, "setrtmp_success"), chatID))
	return telegram.ErrEndGroup
}
