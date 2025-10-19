package handlers

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"
	"github.com/AshokShau/TgMusicBot/pkg/vc"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// queueHandler displays the current playback queue with detailed information.
func queueHandler(m *tg.NewMessage) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	chat := m.Channel
	queue := cache.ChatCache.GetQueue(chatID)
	if len(queue) == 0 {
		_, _ = m.Reply(lang.GetString(langCode, "queue_empty"))
		return nil
	}

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.Reply(lang.GetString(langCode, "queue_no_session"))
		return nil
	}

	current := queue[0]
	playedTime, _ := vc.Calls.PlayedTime(chatID)

	var b strings.Builder
	b.WriteString(fmt.Sprintf(lang.GetString(langCode, "queue_header"), chat.Title))

	b.WriteString(lang.GetString(langCode, "queue_now_playing"))
	b.WriteString(fmt.Sprintf(lang.GetString(langCode, "queue_track_title"), truncate(current.Name, 45)))
	b.WriteString(fmt.Sprintf(lang.GetString(langCode, "queue_requested_by"), current.User))
	b.WriteString(fmt.Sprintf(lang.GetString(langCode, "queue_duration"), cache.SecToMin(current.Duration)))
	b.WriteString(lang.GetString(langCode, "queue_loop"))
	if current.Loop > 0 {
		b.WriteString(lang.GetString(langCode, "queue_loop_on"))
	} else {
		b.WriteString(lang.GetString(langCode, "queue_loop_off"))
	}
	b.WriteString(lang.GetString(langCode, "queue_progress"))
	if playedTime > 0 && playedTime < math.MaxInt {
		b.WriteString(cache.SecToMin(int(playedTime)))
	} else {
		b.WriteString("0:00")
	}
	b.WriteString(" min\n")

	if len(queue) > 1 {
		b.WriteString(fmt.Sprintf(lang.GetString(langCode, "queue_next_up"), len(queue)-1))

		for i, song := range queue[1:] {
			if i >= 14 {
				break
			}
			b.WriteString(strconv.Itoa(i + 1))
			b.WriteString(". <code>")
			b.WriteString(truncate(song.Name, 45))
			b.WriteString("</code> | ")
			b.WriteString(cache.SecToMin(song.Duration))
			b.WriteString(" min\n")
		}

		if len(queue) > 15 {
			b.WriteString(fmt.Sprintf(lang.GetString(langCode, "queue_more_tracks"), len(queue)-15))
		}
	}

	b.WriteString(fmt.Sprintf(lang.GetString(langCode, "queue_total"), len(queue)))

	text := b.String()
	if len(text) > 4096 {
		var sb strings.Builder
		progress := "0:00"
		if playedTime > 0 && playedTime < math.MaxInt {
			progress = cache.SecToMin(int(playedTime))
		}
		sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "queue_short_summary"), chat.Title, truncate(current.Name, 45), progress, cache.SecToMin(current.Duration), len(queue)))
		text = sb.String()
	}

	_, err := m.Reply(text)
	return err
}
