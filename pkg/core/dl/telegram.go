package dl

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// GetMessage retrieves a Telegram message by its URL.
// It supports both public (e.g., https://t.me/ChannelName/1234) and private (e.g., https://t.me/c/12345678/90) URLs.
// It returns the message object or an error if the URL is invalid or the message cannot be fetched.
func GetMessage(client *tg.Client, url string) (*tg.NewMessage, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, errors.New("the provided URL is empty")
	}

	parseTelegramURL := func(input string) (username string, chatID int64, msgID int, isPrivate bool, ok bool) {
		// Regular expression for public channel URLs.
		publicRe := regexp.MustCompile(`^https?://t\.me/([a-zA-Z0-9_]{4,})/(\d+)$`)
		if matches := publicRe.FindStringSubmatch(input); matches != nil {
			id, err := strconv.Atoi(matches[2])
			if err != nil {
				return "", 0, 0, false, false
			}
			return matches[1], 0, id, false, true
		}

		// Regular expression for private or supergroup channel URLs.
		privateRe := regexp.MustCompile(`^https?://t\.me/c/(\d+)/(\d+)$`)
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
		return client.GetMessageByID(chatID, int32(msgID))
	}

	return client.GetMessageByID(username, int32(msgID))
}
