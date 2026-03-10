/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package utils

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

var (
	publicRe  = regexp.MustCompile(`^https?://t\.me/([a-zA-Z0-9_]{4,})/(\d+)$`)
	privateRe = regexp.MustCompile(`^https?://t\.me/c/(\d+)/(\d+)$`)
)

// GetMessage retrieves a Telegram message by its URL.
func GetMessage(client *td.Client, url string) (*td.Message, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, errors.New("the provided URL is empty")
	}

	parseTelegramURL := func(input string) (username string, chatID int64, msgID int, isPrivate bool, ok bool) {
		if matches := publicRe.FindStringSubmatch(input); matches != nil {
			id, err := strconv.Atoi(matches[2])
			if err != nil {
				return "", 0, 0, false, false
			}
			return matches[1], 0, id, false, true
		}

		if matches := privateRe.FindStringSubmatch(input); matches != nil {
			chat, err1 := strconv.ParseInt(matches[1], 10, 64)
			msg, err2 := strconv.Atoi(matches[2])
			if err1 != nil || err2 != nil {
				return "", 0, 0, true, false
			}
			return "", chat, msg, true, true
		}

		return "", 0, 0, false, false
	}

	username, chatID, msgID, isPrivate, ok := parseTelegramURL(url)
	if !ok {
		return nil, errors.New("the provided Telegram URL is invalid")
	}

	if isPrivate {
		link := fmt.Sprintf("https://t.me/c/%d/%d", chatID, msgID)
		info, err := client.GetMessageLinkInfo(link)
		if err != nil {
			slog.Info("failed to get message link info", "error", err)
			return nil, err
		}

		msg := info.Message
		if msg == nil {
			msg, err = client.GetMessage(info.Message.ChatId, info.Message.Id)
			if err != nil {
				slog.Info("failed to get message", "error", err)
				return nil, err
			}
			return msg, nil
		}

		return msg, nil
	}

	link := fmt.Sprintf("https://t.me/%s/%d", username, msgID)
	info, err := client.GetMessageLinkInfo(link)
	if err != nil {
		slog.Info("failed to get message link info", "error", err)
		return nil, err
	}

	msg := info.Message
	if msg == nil {
		msg, err = client.GetMessage(info.Message.ChatId, info.Message.Id)
		if err != nil {
			slog.Info("failed to get message", "error", err)
			return nil, err
		}
		return msg, nil
	}

	return msg, nil
}
