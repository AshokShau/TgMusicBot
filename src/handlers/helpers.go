/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"github.com/amarnathcjd/gogram/telegram"
)

// getUrl gets a URL from a message.
// It takes a telegram.NewMessage object and a boolean indicating whether it is a reply.
// It returns the URL from the message.
func getUrl(m *telegram.NewMessage, isReply bool) string {
	text := m.Text()
	entities := m.Message.Entities
	if isReply {
		reply, err := m.GetReplyMessage()
		if err == nil && reply != nil {
			text = reply.Text()
			entities = reply.Message.Entities
		}
	}

	if entities == nil || len(entities) == 0 {
		return ""
	}

	for _, entity := range entities {
		switch e := entity.(type) {
		case *telegram.MessageEntityTextURL:
			return e.URL
		case *telegram.MessageEntityURL:
			url := text[e.Offset : e.Offset+e.Length]
			return url
		default:
			logger.Debug("Ignoring entity type: %T", e)
		}
	}

	return ""
}

// isValidMedia checks if a message contains valid media.
// It takes a telegram.NewMessage object as input.
// It returns true if the message contains valid media, otherwise false.
func isValidMedia(reply *telegram.NewMessage) bool {
	if !reply.IsMedia() {
		return false
	}

	if reply.Audio() == nil && reply.Video() == nil && reply.Document() == nil {
		return false
	}

	return true
}

// coalesce returns the first non-empty string.
// It takes two strings as input.
// It returns the first non-empty string.
func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// truncate truncates a string to a maximum length.
// It takes a string and a maximum length as input.
// It returns the truncated string.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// getMemberCount gets the number of members in a chat.
// It returns the count and any error encountered.
func getMemberCount(client *telegram.Client, chatID int64) (int, error) {
	// Get chat info
	chat, err := client.GetChat(chatID)
	if err != nil {
		return 0, err
	}

	// For channels and groups
	switch c := chat.(type) {
	case *telegram.Channel:
		// Try to get from full channel info
		fullChat, err := client.ChannelsGetFullChannel(&telegram.InputChannelObj{
			ChannelID:  c.ID,
			AccessHash: c.AccessHash,
		})
		if err != nil {
			return 0, err
		}
		if fullChat != nil && fullChat.FullChat != nil {
			if fc, ok := fullChat.FullChat.(*telegram.ChannelFull); ok {
				return int(fc.ParticipantsCount), nil
			}
		}
	case *telegram.ChatObj:
		return int(c.ParticipantsCount), nil
	}

	return 0, nil
}
