/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
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
	"ashokshau/tgmusic/src/utils"

	"github.com/amarnathcjd/gogram/telegram"
)

// pingHandler handles the /ping command.
func pingHandler(m *telegram.NewMessage) error {
	start := time.Now()
	updateLag := time.Since(time.Unix(int64(m.Date()), 0)).Milliseconds()

	msg, err := m.Reply(utils.EmojiSpeed + " Pinging...")
	if err != nil {
		return err
	}

	latency := time.Since(start).Milliseconds()
	uptime := time.Since(startTime).Truncate(time.Second)
	senders := m.Client.GetExportedSendersStatus()
	response := fmt.Sprintf(
		"%s <b>System Performance Metrics</b>\n\n"+
			"%s <b>Bot Latency:</b> <code>%d ms</code>\n"+
			"%s <b>Uptime:</b> <code>%s</code>\n"+
			"%s <b>Update Lag:</b> <code>%d ms</code>\n"+
			"%s <b>Go Routines:</b> <code>%d</code>\n"+
			"%s <b>Senders:</b> <code>%d</code>\n",
		utils.EmojiGear,
		utils.EmojiSpeed, latency,
		utils.EmojiHourglass, uptime,
		utils.EmojiSearch, updateLag,
		utils.EmojiGear, runtime.NumGoroutine(),
		utils.EmojiTelegram, senders,
	)

	_, err = msg.Edit(response)
	return err
}

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

	response := fmt.Sprintf("%s Hello %s!\n\nI am %s, a fast and powerful music player for Telegram.\n\n<b>Supported Platforms:</b> YouTube, Spotify, Apple Music, SoundCloud.\n\nClick the <b>Help</b> button below for more information.", utils.EmojiMusic, m.Sender.FirstName, bot.FirstName)
	_, err := m.Reply(response, &telegram.SendOptions{
		ReplyMarkup: core.AddMeMarkup(m.Client.Me().Username),
	})

	return err
}
