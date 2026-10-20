/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"errors"
	"strings"
	"unicode/utf16"

	td "github.com/AshokShau/gotdbot"
)

func getRepliedMessage(c *td.Client, m *td.Message) (*td.Message, error) {
	if m == nil {
		return nil, errors.New("missing reply message")
	}

	message, err := m.GetRepliedMessage(c)
	if err == nil && message != nil {
		return message, nil
	}

	reply, ok := m.ReplyTo.(*td.MessageReplyToMessage)
	if !ok || reply == nil || reply.MessageId == 0 {
		return nil, errors.New("missing reply message")
	}

	chatId := reply.ChatId
	if chatId == 0 {
		chatId = m.ChatId
	}

	message, err = c.GetMessage(chatId, reply.MessageId)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, errors.New("reply message not found")
	}
	return message, nil
}

func getUrl(m *td.Message) string {
	if m == nil {
		return ""
	}
	text := utf16.Encode([]rune(m.GetText()))
	entities := m.GetEntities()

	if entities == nil || len(entities) == 0 {
		return ""
	}

	for _, entity := range entities {
		switch t := entity.Type.(type) {

		case *td.TextEntityTypeUrl:
			start := entity.Offset
			end := entity.Offset + entity.Length
			if start >= 0 && end > start && int(end) <= len(text) {
				return string(utf16.Decode(text[start:end]))
			}

		case *td.TextEntityTypeTextUrl:
			return t.Url
		}
	}

	return ""
}

func isValidMedia(reply *td.Message) bool {
	if reply == nil || reply.Content == nil {
		return false
	}

	switch msg := reply.Content.(type) {

	case *td.MessageAudio,
		*td.MessageVoiceNote,
		*td.MessageVideo,
		*td.MessageVideoNote:
		return true

	case *td.MessageDocument:
		if msg.Document == nil {
			return false
		}
		mime := strings.ToLower(msg.Document.MimeType)
		if strings.HasPrefix(mime, "audio/") || strings.HasPrefix(mime, "video/") {
			return true
		}

		return false
	}

	return false
}

func getFile(m *td.Message) (*td.File, string) {
	if m == nil || m.Content == nil {
		return nil, ""
	}

	switch content := m.Content.(type) {
	case *td.MessageAudio:
		return content.Audio.Audio, content.Audio.Title
	case *td.MessageVoiceNote:
		return content.VoiceNote.Voice, "voice_note.ogg"
	case *td.MessageVideo:
		return content.Video.Video, content.Video.FileName
	case *td.MessageVideoNote:
		return content.VideoNote.Video, "video_note.mp4"
	case *td.MessageDocument:
		return content.Document.Document, content.Document.FileName
	default:
		return nil, ""
	}
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
