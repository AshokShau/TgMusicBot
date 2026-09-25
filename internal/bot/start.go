/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tgmusic/internal/config"
	"ashokshau/tgmusic/internal/utils"
	"fmt"
	"runtime"
	"time"

	td "github.com/AshokShau/gotdbot"
)

func pingHandler(c *td.Client, m *td.Message) error {
	deleteCmd(c, m)

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

func startHandler(c *td.Client, m *td.Message) error {
	chatID := m.ChatId
	go storeChatToDB(chatID)

	deleteCmd(c, m)

	if m.IsPrivate() {
		response := fmt.Sprintf(
			"<img src=\"%s\"/>\n"+
				"<h3>Welcome, %s!</h3>\n"+
				"<p><b>%s</b> lets you stream high-quality music and video directly in Telegram voice and video chats.</p>\n\n"+
				"<p><b>Supported platforms:</b> YouTube, Spotify, Apple Music, SoundCloud, Deezer, Twitch, and many more.</p>\n\n"+
				"<p>Use the buttons below to add the bot to your group or explore the available commands.</p>",
			config.StartImg,
			firstName(c, m),
			c.Me.FirstName,
		)

		richMessage := &td.InputRichMessage{
			Source: &td.RichMessageSourceHtml{
				Text: response,
			},
		}

		_, err := m.ReplyRichMessage(c, richMessage, &td.SendTextMessageOpts{
			ReplyMarkup: utils.AddMeMarkup(c.Me.Usernames.EditableUsername),
		})

		return err
	}

	uptime := getFormattedDuration(time.Since(startTime))
	htmlText := fmt.Sprintf(
		"<h3>%s is ready!</h3>\n"+
			"<p><b>Uptime:</b> <code>%s</code></p>\n"+
			"<p><i>A feature-rich music bot for your group video chats. Play your favorite tracks seamlessly.</i></p>\n\n"+
			"<p><tg-button type=\"url\" url=\"%s\">Updates</tg-button> <tg-button type=\"url\" url=\"%s\">Group</tg-button></p>",
		c.Me.FirstName,
		uptime,
		config.SupportChannel,
		config.SupportGroup,
	)

	richMessage := &td.InputRichMessage{
		Source: &td.RichMessageSourceHtml{
			Text: htmlText,
		},
	}

	_, err := m.ReplyRichMessage(c, richMessage, nil)
	return err
}
