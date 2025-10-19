package handlers

import (
	"fmt"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"
	"github.com/amarnathcjd/gogram/telegram"
)

func langHandler(m *telegram.NewMessage) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	_, err := m.Reply(lang.GetString(langCode, "choose_lang"), telegram.SendOptions{
		ReplyMarkup: core.LanguageKeyboard(),
	})
	return err
}

func setLangCallbackHandler(c *telegram.CallbackQuery) error {
	parts := strings.SplitN(c.DataString(), "_", 2)
	if len(parts) < 2 {
		return nil
	}
	langCode := parts[1]

	// Validate that the language code is supported
	supportedLangs := lang.GetAvailableLangs()
	isValidLang := false
	for _, supportedLang := range supportedLangs {
		if supportedLang == langCode {
			isValidLang = true
			break
		}
	}

	if !isValidLang {
		_, err := c.Answer("❌ Unsupported language code", &telegram.CallbackOptions{Alert: true})
		return err
	}

	chatID, _ := getPeerId(c.Client, c.ChatID)
	ctx, cancel := db.Ctx()
	defer cancel()

	if c.IsPrivate() {
		_ = db.Instance.SetUserLang(ctx, chatID, langCode)
	} else {
		admins, err := cache.GetAdmins(c.Client, chatID, false)
		if err != nil {
			return err
		}
		var isAdmin bool
		for _, admin := range admins {
			if admin.User.ID == c.Sender.ID {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			_, err := c.Answer(lang.GetString(langCode, "lang_no_permission"), &telegram.CallbackOptions{Alert: true})
			return err
		}

		_ = db.Instance.SetChatLang(ctx, chatID, langCode)
	}

	_, _ = c.Answer(fmt.Sprintf(lang.GetString(langCode, "lang_updated"), langCode), &telegram.CallbackOptions{Alert: true})
	_, err := c.Edit(fmt.Sprintf(lang.GetString(langCode, "lang_changed"), langCode))
	return err
}
