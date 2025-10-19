package handlers

import (
	"fmt"
	"strconv"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"

	"github.com/amarnathcjd/gogram/telegram"
)

// loopHandler handles the /loop command.
func loopHandler(m *telegram.NewMessage) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.Reply(lang.GetString(langCode, "no_track_playing"))
		return err
	}

	args := m.Args()
	if args == "" {
		_, err := m.Reply(lang.GetString(langCode, "loop_usage"))
		return err
	}

	argsInt, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.Reply(lang.GetString(langCode, "loop_invalid_count"))
		return nil
	}

	if argsInt < 0 || argsInt > 10 {
		_, err = m.Reply(lang.GetString(langCode, "loop_out_of_range"))
		return err
	}

	cache.ChatCache.SetLoopCount(chatID, argsInt)
	var action string
	if argsInt == 0 {
		action = lang.GetString(langCode, "loop_disabled")
	} else {
		action = fmt.Sprintf(lang.GetString(langCode, "loop_set"), argsInt)
	}

	_, err = m.Reply(fmt.Sprintf(lang.GetString(langCode, "loop_status_changed"), action, m.Sender.FirstName))
	return err
}
