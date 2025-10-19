package vc

import (
	"fmt"

	"github.com/AshokShau/TgMusicBot/pkg/config"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// sendLogger sends a formatted log message to the designated logger chat.
// It includes details about the song being played, such as its title, duration, and the user who requested it.
func sendLogger(client *tg.Client, chatID int64, song *cache.CachedTrack) {
	if chatID == 0 || song == nil || chatID == config.Conf.LoggerId {
		return
	}

	text := fmt.Sprintf(
		"<b>A song is playing</b> in <code>%d</code>\n\n‣ <b>Title:</b> <a href='%s'>%s</a>\n‣ <b>Duration:</b> %s\n‣ <b>Requested by:</b> %s\n‣ <b>Platform:</b> %s\n‣ <b>Is Video:</b> %t",
		chatID,
		song.URL,
		song.Name,
		cache.SecToMin(song.Duration),
		song.User,
		song.Platform,
		song.IsVideo,
	)

	_, err := client.SendMessage(config.Conf.LoggerId, text, &tg.SendOptions{LinkPreview: false})
	if err != nil {
		gologging.WarnF("[sendLogger] Failed to send the message: %v", err)
	}
}
