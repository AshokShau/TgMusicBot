/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/config"
	"fmt"
	"html"
	"strings"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/utils"
	"ashokshau/tgmusic/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// activeVcHandler handles the /activevc command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func activeVcHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	activeChats := cache.ChatCache.GetActiveChats()

	var sb strings.Builder
	sb.WriteString("<h3>🎵 Active Voice Chats</h3>")

	if len(activeChats) == 0 {
		sb.WriteString("<blockquote><b>🔇 No Active Chats:</b> There are currently no active voice or video chats.</blockquote>")
	} else {
		sb.WriteString(fmt.Sprintf("<p>There are currently <b>%d</b> active voice/video chat(s) running.</p>", len(activeChats)))
		sb.WriteString("<details>")
		sb.WriteString("<summary><b>📊 Click to Show Active Chats Table</b></summary>")
		sb.WriteString("<br>")
		sb.WriteString("<table bordered striped>")
		sb.WriteString("<tr>")
		sb.WriteString("<th align='center'><b>#</b></th>")
		sb.WriteString("<th align='center'><b>Chat ID</b></th>")
		sb.WriteString("<th align='center'><b>Queue</b></th>")
		sb.WriteString("<th align='left'><b>Now Playing Track Info</b></th>")
		sb.WriteString("<th align='center'><b>Type</b></th>")
		sb.WriteString("</tr>")

		for i, chatID := range activeChats {
			queueLength := cache.ChatCache.GetQueueLength(chatID)
			currentSong := cache.ChatCache.GetPlayingTrack(chatID)

			var trackLink, trackType string
			if currentSong != nil {
				trackName := html.EscapeString(currentSong.Name)
				trackURL := html.EscapeString(currentSong.URL)
				if trackURL == "" {
					trackURL = "https://t.me/FallenProjects"
				}
				durStr := utils.SecToMin(currentSong.Duration)
				trackLink = fmt.Sprintf("<a href='%s'>%s</a> (%s)", trackURL, trackName, durStr)
				if currentSong.IsVideo {
					trackType = "<tg-emoji emoji-id='5368324170671202286'>🎥</tg-emoji> Video"
				} else {
					trackType = "<tg-emoji emoji-id='5368324170671202286'>🎵</tg-emoji> Audio"
				}
			} else {
				trackLink = "<i>🔇 No song playing.</i>"
				trackType = "<tg-emoji emoji-id='5368324170671202286'>🎵</tg-emoji> Audio"
			}

			sb.WriteString("<tr>")
			sb.WriteString(fmt.Sprintf("<td align='center'>%d</td>", i+1))
			sb.WriteString(fmt.Sprintf("<td align='center'><code>%d</code></td>", chatID))
			sb.WriteString(fmt.Sprintf("<td align='center'>%d</td>", queueLength))
			sb.WriteString(fmt.Sprintf("<td align='left'>%s</td>", trackLink))
			sb.WriteString(fmt.Sprintf("<td align='center'>%s</td>", trackType))
			sb.WriteString("</tr>")
		}
		sb.WriteString("</table>")
		sb.WriteString("</details>")
		sb.WriteString("<br>")
	}

	richMessage := &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: sb.String()}}
	_, err := m.ReplyRichMessage(c, richMessage, nil)
	return err
}

// Handles the /clearass command to remove all assistant assignments
func clearAssistantsHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	done, err := db.Instance.ClearAllAssistants()
	if err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("failed to clear assistants: %s", err.Error()), nil)
		return td.EndGroups
	}

	_, err = m.ReplyText(c, fmt.Sprintf("Removed assistant from %d chats", done), nil)
	return err
}

// Handles the /leaveall command to leave all chats
func leaveAllHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	reply, err := m.ReplyText(c, "Assistant is leaving all chats...", nil)
	if err != nil {
		return err
	}

	leftCount, err := vc.Calls.LeaveAll()
	if err != nil {
		_, _ = reply.EditText(c, fmt.Sprintf("Failed to leave all chats: %s", err.Error()), nil)
		return err
	}

	_, err = reply.EditText(c, fmt.Sprintf("Assistant's Left %d chats", leftCount), nil)
	return err
}

// Handles the /logger command to toggle logger status
func loggerHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	if config.LoggerId == 0 {
		_, _ = m.ReplyText(c, "Please set LOGGER_ID in .env first.", nil)
		return td.EndGroups
	}

	loggerStatus := db.Instance.GetLoggerStatus()
	args := strings.ToLower(Args(m))
	if len(args) == 0 {
		_, _ = m.ReplyText(c, fmt.Sprintf("Usage: /logger [enable|disable|on|off]\nCurrent status: %t", loggerStatus), nil)
		return td.EndGroups
	}

	switch args {
	case "enable", "on":
		_ = db.Instance.SetLoggerStatus(true)
		_, _ = m.ReplyText(c, "Logger Enabled", nil)
	case "disable", "off":
		_ = db.Instance.SetLoggerStatus(false)
		_, _ = m.ReplyText(c, "Logger disabled", nil)
	default:
		_, _ = m.ReplyText(c, "Invalid argument. Use 'enable', 'disable', 'on', or 'off'.", nil)
	}

	return td.EndGroups
}
