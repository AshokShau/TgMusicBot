package handlers

import (
	"fmt"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

// getPeerId gets the peer ID from a chat ID.
// It takes a telegram client and a chat ID as input.
// It returns the peer ID and an error if any.
func getPeerId(c *telegram.Client, chatId any) (int64, error) {
	peer, err := c.ResolvePeer(chatId)
	if err != nil {
		gologging.WarnF("failed to resolve Peer for %d", chatId)
		return 0, err
	}

	switch p := peer.(type) {
	case *telegram.InputPeerUser:
		return p.UserID, nil
	case *telegram.InputPeerChat:
		return -p.ChatID, nil
	case *telegram.InputPeerChannel:
		return -1000000000000 - p.ChannelID, nil
	default:
		return 0, fmt.Errorf("unsupported peer type %T", p)
	}
}

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
			gologging.DebugF("Ignoring entity type: %T", e)
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
