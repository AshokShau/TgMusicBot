package handlers

import (
	"fmt"
	"time"

	"github.com/AshokShau/TgMusicBot/pkg/core"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"

	"github.com/amarnathcjd/gogram/telegram"
)

// pingHandler handles the /ping command.
func pingHandler(m *telegram.NewMessage) error {
	start := time.Now()
	msg, err := m.Reply("⏱️ Pinging...")
	if err != nil {
		return err
	}
	latency := time.Since(start).Milliseconds()
	uptime := time.Since(startTime).Truncate(time.Second)

	ctx, cancel := db.Ctx()
	defer cancel()

	chatID, _ := getPeerId(m.Client, m.ChatID())
	langCode := db.Instance.GetLang(ctx, chatID)
	response := fmt.Sprintf(lang.GetString(langCode, "ping_text"), latency, uptime)
	_, err = msg.Edit(response)
	return err
}

// startHandler handles the /start command.
func startHandler(m *telegram.NewMessage) error {
	bot := m.Client.Me()
	chatID, _ := getPeerId(m.Client, m.ChatID())

	if m.IsPrivate() {
		go func(chatID int64) {
			ctx, cancel := db.Ctx()
			defer cancel()
			_ = db.Instance.AddUser(ctx, chatID)
		}(chatID)
	} else {
		go func(chatID int64) {
			ctx, cancel := db.Ctx()
			defer cancel()
			_ = db.Instance.AddChat(ctx, chatID)
		}(chatID)
	}

	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)

	response := fmt.Sprintf(lang.GetString(langCode, "start_text"), m.Sender.FirstName, bot.FirstName)
	_, err := m.Reply(response, telegram.SendOptions{
		ReplyMarkup: core.AddMeMarkup(m.Client.Me().Username),
	})

	return err
}
