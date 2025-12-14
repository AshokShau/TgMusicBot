/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"strconv"
)

func parseTelegramURL(input string) (string, int, bool) {
	if input == "" {
		return "", 0, false
	}

	match := telegramURLRegex.FindStringSubmatch(input)
	if match == nil {
		return "", 0, false
	}

	id, err := strconv.Atoi(match[2])
	if err != nil {
		return "", 0, false
	}

	return match[1], id, true
}
