package handlers

import (
	"fmt"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"

	"github.com/amarnathcjd/gogram/telegram"
)

// activeVcHandler handles the /activevc command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func activeVcHandler(m *telegram.NewMessage) error {
	activeChats := cache.ChatCache.GetActiveChats()
	if len(activeChats) == 0 {
		_, err := m.Reply("No active chats found.")
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎵 <b>Active Voice Chats</b> (%d):\n\n", len(activeChats)))

	for _, chatID := range activeChats {
		queueLength := cache.ChatCache.GetQueueLength(chatID)
		currentSong := cache.ChatCache.GetPlayingTrack(chatID)

		var songInfo string
		if currentSong != nil {
			songInfo = fmt.Sprintf(
				"🎶 <b>Now Playing:</b> <a href='%s'>%s</a> (%ds)",
				currentSong.URL,
				currentSong.Name,
				currentSong.Duration,
			)
		} else {
			songInfo = "🔇 No song playing."
		}

		sb.WriteString(fmt.Sprintf(
			"➤ <b>Chat ID:</b> <code>%d</code>\n📌 <b>Queue Size:</b> %d\n%s\n\n",
			chatID,
			queueLength,
			songInfo,
		))
	}

	text := sb.String()
	if len(text) > 4096 {
		text = fmt.Sprintf("🎵 <b>Active Voice Chats</b> (%d)", len(activeChats))
	}

	_, err := m.Reply(text, telegram.SendOptions{LinkPreview: false})
	if err != nil {
		return err
	}

	return nil
}
