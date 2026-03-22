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
	"strings"
	"time"

	"github.com/AshokShau/gotdbot"
)

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

// plural returns the unit with correct singular/plural form.
func plural(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, unit)
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// getFormattedDuration returns a human-readable string for the given duration.
func getFormattedDuration(diff time.Duration) string {
	totalSeconds := int(diff.Seconds())

	months := totalSeconds / (30 * 24 * 3600)
	remaining := totalSeconds % (30 * 24 * 3600)

	weeks := remaining / (7 * 24 * 3600)
	remaining = remaining % (7 * 24 * 3600)

	days := remaining / (24 * 3600)
	remaining = remaining % (24 * 3600)

	hours := remaining / 3600
	remaining = remaining % 3600

	minutes := remaining / 60
	seconds := remaining % 60

	var parts []string

	if months > 0 {
		parts = append(parts, plural(months, "month"))
	}
	if weeks > 0 {
		parts = append(parts, plural(weeks, "week"))
	}
	if days > 0 {
		parts = append(parts, plural(days, "day"))
	}
	if hours > 0 {
		parts = append(parts, plural(hours, "hour"))
	}
	if minutes > 0 {
		parts = append(parts, plural(minutes, "minute"))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, plural(seconds, "second"))
	}

	return strings.Join(parts, " ")
}
