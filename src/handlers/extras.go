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
	"ashokshau/tgmusic/src/utils"
	"strconv"
	"strings"

	"github.com/AshokShau/gotdbot"
)

func parseTelegramURL(input string) (string, int, bool) {
	if input == "" {
		return "", 0, false
	}

	match := utils.TelegramMessageRegex.FindStringSubmatch(input)
	if match == nil {
		return "", 0, false
	}

	id, err := strconv.Atoi(match[2])
	if err != nil {
		return "", 0, false
	}

	return match[1], id, true
}

func Args(m *gotdbot.Message) string {
	Messages := strings.Split(m.Text(), " ")
	if len(Messages) < 2 {
		return ""
	}
	return strings.TrimSpace(strings.Join(Messages[1:], " "))
}

// isDev checks if the user is a developer.
// It returns true if the user is a developer, otherwise false.
func isDev(ctx *gotdbot.Context) bool {
	m := ctx.EffectiveMessage

	for _, dev := range config.Conf.DEVS {
		if dev == m.SenderID() {
			return true
		}
	}

	return false
}

func SenderID(sender gotdbot.MessageSender) int64 {
	switch s := sender.(type) {
	case *gotdbot.MessageSenderUser:
		return s.UserId
	case *gotdbot.MessageSenderChat:
		return s.ChatId
	default:
		return 0
	}
}
