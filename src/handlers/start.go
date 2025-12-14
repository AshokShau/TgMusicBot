/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"fmt"
	"runtime"
	"time"

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"

	"github.com/amarnathcjd/gogram/telegram"
)

// pingHandler handles the /ping command.
func pingHandler(m *telegram.NewMessage) error {
	start := time.Now()
	updateLag := time.Since(time.Unix(int64(m.Date()), 0)).Milliseconds()

	msg, err := m.Reply("⏱️ Pinging...")
	if err != nil {
		return err
	}

	latency := time.Since(start).Milliseconds()
	uptime := time.Since(startTime).Truncate(time.Second)
	senders := m.Client.GetExportedSendersStatus()
	response := fmt.Sprintf(
		"<b>📊 System Performance Metrics</b>\n\n"+
			"⏱️ <b>Bot Latency:</b> <code>%d ms</code>\n"+
			"🕒 <b>Uptime:</b> <code>%s</code>\n"+
			"📩 <b>Update Lag:</b> <code>%d ms</code>\n"+
			"⚙️ <b>Go Routines:</b> <code>%d</code>\n"+
			"📨 <b>Senders:</b> <code>%d</code>\n",
		latency, uptime, updateLag, runtime.NumGoroutine(), senders,
	)

	_, err = msg.Edit(response)
	return err
}

// StartImageURL is the URL of the image to send with the /start command.
// Change this to your desired image URL.
const StartImageURL = "https://files.catbox.moe/svrc2j.jpg"

// startHandler handles the /start command.
func startHandler(m *telegram.NewMessage) error {
	bot := m.Client.Me()
	chatID := m.ChannelID()

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

	// Get connected groups and users count
	chats, _ := db.Instance.GetAllChats(ctx)
	users, _ := db.Instance.GetAllUsers(ctx)
	groupCount := len(chats)
	userCount := len(users)

	response := fmt.Sprintf(lang.GetString(langCode, "start_text"), m.Sender.FirstName, bot.FirstName)
	response += fmt.Sprintf("\n\n<b>📊 Stats:</b>\n├ 👥 Users: <code>%d</code>\n└ 💬 Groups: <code>%d</code>", userCount, groupCount)

	// Send photo with caption
	_, err := m.Client.SendMedia(chatID, StartImageURL, &telegram.MediaOptions{
		Caption:     response,
		ReplyMarkup: core.AddMeMarkup(m.Client.Me().Username),
	})

	return err
}
