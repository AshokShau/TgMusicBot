package handlers

import (
	"math"
	"strconv"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/vc"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// queueHandler displays the current playback queue with detailed information.
func queueHandler(m *tg.NewMessage) error {
	chatId, _ := getPeerId(m.Client, m.ChatID())

	chat := m.Channel
	queue := cache.ChatCache.GetQueue(chatId)
	if len(queue) == 0 {
		_, _ = m.Reply("📭 The queue is currently empty.")
		return nil
	}

	if !cache.ChatCache.IsActive(chatId) {
		_, _ = m.Reply("⏸ There is no active playback session.")
		return nil
	}

	current := queue[0]
	playedTime, _ := vc.Calls.PlayedTime(chatId)

	var b strings.Builder
	b.WriteString("<b>🎧 Queue for ")
	b.WriteString(chat.Title)
	b.WriteString("</b>\n\n")

	b.WriteString("<b>▶️ Now Playing:</b>\n")
	b.WriteString("├ <b>Title:</b> <code>")
	b.WriteString(truncate(current.Name, 45))
	b.WriteString("</code>\n")
	b.WriteString("├ <b>Requested by:</b> ")
	b.WriteString(current.User)
	b.WriteString("\n├ <b>Duration:</b> ")
	b.WriteString(cache.SecToMin(current.Duration))
	b.WriteString(" min\n")
	b.WriteString("├ <b>Loop:</b> ")
	if current.Loop > 0 {
		b.WriteString("🔁 On\n")
	} else {
		b.WriteString("➡️ Off\n")
	}
	b.WriteString("└ <b>Progress:</b> ")
	if playedTime > 0 && playedTime < math.MaxInt {
		b.WriteString(cache.SecToMin(int(playedTime)))
	} else {
		b.WriteString("0:00")
	}
	b.WriteString(" min\n")

	if len(queue) > 1 {
		b.WriteString("\n<b>⏭ Next Up (")
		b.WriteString(strconv.Itoa(len(queue) - 1))
		b.WriteString("):</b>\n")

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
			b.WriteString("...and ")
			b.WriteString(strconv.Itoa(len(queue) - 15))
			b.WriteString(" more track(s)\n")
		}
	}

	b.WriteString("\n<b>📊 Total:</b> ")
	b.WriteString(strconv.Itoa(len(queue)))
	b.WriteString(" track(s) in the queue")

	text := b.String()
	if len(text) > 4096 {
		var sb strings.Builder
		sb.WriteString("<b>🎧 Queue for ")
		sb.WriteString(chat.Title)
		sb.WriteString("</b>\n\n<b>▶️ Now Playing:</b>\n├ <code>")
		sb.WriteString(truncate(current.Name, 45))
		sb.WriteString("</code>\n└ ")
		if playedTime > 0 && playedTime < math.MaxInt {
			sb.WriteString(cache.SecToMin(int(playedTime)))
		} else {
			sb.WriteString("0:00")
		}
		sb.WriteString("/")
		sb.WriteString(cache.SecToMin(current.Duration))
		sb.WriteString(" min\n\n<b>📊 Total:</b> ")
		sb.WriteString(strconv.Itoa(len(queue)))
		sb.WriteString(" track(s) in the queue")
		text = sb.String()
	}

	_, err := m.Reply(text)
	return err
}
