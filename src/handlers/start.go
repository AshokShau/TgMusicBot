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

	td "github.com/AshokShau/gotdbot"
)

// pingHandler handles the /ping command.
func pingHandler(c *td.Client, ctx *td.Context) error {
	m := ctx.EffectiveMessage
	start := time.Now()

	msg, err := m.ReplyText(c, "Pinging… please wait…", nil)
	if err != nil {
		return err
	}

	latency := time.Since(start).Milliseconds()
	uptime := getFormattedDuration(time.Since(startTime))

	response := fmt.Sprintf(
		"<b>📊 System Performance Metrics</b>\n\n"+
			"<b>Bot Latency:</b> <code>%d ms</code>\n"+
			"<b>Uptime:</b> <code>%s</code>\n"+
			"<b>Go Routines:</b> <code>%d</code>\n",
		latency, uptime, runtime.NumGoroutine(),
	)

	_, err = msg.EditText(c, response, &td.EditTextMessageOpts{ParseMode: "HTML"})
	return err
}

// startHandler handles the /start command.
func startHandler(c *td.Client, ctx *td.Context) error {
	chatID := ctx.EffectiveChatId
	m := ctx.EffectiveMessage

	if m.IsPrivate() {
		go func(chatID int64) {
			_ = db.Instance.AddUser(chatID)
		}(chatID)
	} else {
		go func(chatID int64) {
			_ = db.Instance.AddChat(chatID)
		}(chatID)
	}

	response := fmt.Sprintf(
		"Hey %s,\nThis is %s !\n\nA music player bot with some awesome and useful features.\n<b>Supported Platforms:</b> YouTube, Spotify, Apple Music, SoundCloud.\n\n<b><i>Click on the help button for more info.</i></b>",
		firstName(c, m),
		c.Me.FirstName,
	)

	_, err := m.ReplyText(c, response, &td.SendTextMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: core.AddMeMarkup(c.Me.Usernames.EditableUsername),
	})

	return err
}
