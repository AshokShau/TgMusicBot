package messaging

import (
	"ashokshau/tgmusic/internal/db"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func IsGroup(chat *td.Chat) bool {
	if chat == nil {
		return false
	}
	switch kind := chat.Type.(type) {
	case *td.ChatTypeBasicGroup:
		return true
	case *td.ChatTypeSupergroup:
		return !kind.IsChannel
	default:
		return false
	}
}

func Silent(chatID int64, explicit bool) bool {
	if chatID >= 0 {
		return false
	}
	if explicit {
		return true
	}
	if db.Instance == nil {
		return false
	}
	settings, err := db.Instance.GetCleanSettings(chatID)
	return err != nil || settings.Silent
}

func SupportsSilence(command string) bool {
	switch strings.ToLower(command) {
	case "play", "p", "fplay", "fp", "vplay", "v", "fvplay", "fvp", "mute", "unmute", "skip", "stop", "end", "pause", "resume",
		"seek", "loop", "remove", "mix", "queue":
		return true
	default:
		return false
	}
}

func command(m *td.Message) string {
	if m == nil {
		return ""
	}
	fields := strings.Fields(m.Text())
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return ""
	}
	return strings.ToLower(strings.Split(strings.TrimPrefix(fields[0], "/"), "@")[0])
}

func SilentCommand(m *td.Message) bool {
	cmd := command(m)
	return m != nil && m.ChatId < 0 && strings.HasPrefix(cmd, "s") && SupportsSilence(strings.TrimPrefix(cmd, "s"))
}

func Quiet(m *td.Message) bool {
	cmd := command(m)
	if SupportsSilence(cmd) || SilentCommand(m) {
		return Silent(m.ChatId, SilentCommand(m))
	}
	return false
}
func Reply(c *td.Client, m *td.Message, text string, opts *td.SendMessageOpts) (*td.Message, error) {
	if Quiet(m) {
		return nil, nil
	}
	return m.ReplyText(c, text, opts)
}

func Send(c *td.Client, chatID int64, text string, opts *td.SendMessageOpts, quiet bool) (*td.Message, error) {
	if Silent(chatID, quiet) {
		return &td.Message{ChatId: chatID}, nil
	}
	return c.SendMessage(chatID, text, opts)
}

// FinishProgress removes a play command's visible progress on silent success,
// or replaces it with the final player/queue message for ordinary playback.
func FinishProgress(c *td.Client, m *td.Message, text string, opts *td.EditMessageTextOpts, explicit bool) (*td.Message, error) {
	if m == nil || m.Id == 0 {
		return nil, nil
	}

	return finishProgress(Silent(m.ChatId, explicit), func() error {
		err := m.Delete(c, &td.DeleteMessagesOpts{Revoke: true})
		if err != nil {
			c.Logger.Warn("failed to delete playback progress", "chat_id", m.ChatId)
		}
		return err
	}, func() (*td.Message, error) {
		return m.EditText(c, text, opts)
	})
}

func finishProgress(silent bool, deleteMessage func() error, editMessage func() (*td.Message, error)) (*td.Message, error) {
	if silent {
		// A failed progress cleanup must not report successful playback as failed.
		_ = deleteMessage()
		return nil, nil
	}
	return editMessage()
}
