package handlers

import (
	"fmt"
	"tgmusic/pkg/core"
	"tgmusic/pkg/core/db"
	"tgmusic/pkg/pool"
	"time"

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
	response := fmt.Sprintf(
		"<b>📊 System Performance Metrics</b>\n\n"+
			"⏱️ <b>Bot Latency:</b> <code>%d ms</code>\n"+
			"🕒 <b>Uptime:</b> <code>%s</code>",
		latency, uptime,
	)
	_, err = msg.Edit(response)
	return err
}

// startHandler handles the /start command.
func startHandler(m *telegram.NewMessage) error {
	bot := m.Client.Me()
	chatID, _ := getPeerId(m.Client, m.ChatID())

	if m.IsPrivate() {
		pool.Submit(func() {
			ctx, cancel := db.Ctx()
			defer cancel()
			_ = db.Instance.AddUser(ctx, chatID)
		})
	} else {
		pool.Submit(func() {
			ctx, cancel := db.Ctx()
			defer cancel()
			_ = db.Instance.AddChat(ctx, chatID)
		})
	}

	response := fmt.Sprintf(startText, m.Sender.FirstName, bot.FirstName)
	_, err := m.Reply(response, telegram.SendOptions{
		ReplyMarkup: core.AddMeMarkup(m.Client.Me().Username),
	})

	return err
}
