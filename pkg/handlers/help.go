package handlers

import (
	"fmt"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"

	"github.com/amarnathcjd/gogram/telegram"
)

func getHelpCategories(langCode string) map[string]struct {
	Title   string
	Content string
	Markup  *telegram.ReplyInlineMarkup
} {
	return map[string]struct {
		Title   string
		Content string
		Markup  *telegram.ReplyInlineMarkup
	}{
		"help_user": {
			Title:   lang.GetString(langCode, "help_user_title"),
			Content: lang.GetString(langCode, "help_user_content"),
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_admin": {
			Title:   lang.GetString(langCode, "help_admin_title"),
			Content: lang.GetString(langCode, "help_admin_content"),
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_devs": {
			Title:   lang.GetString(langCode, "help_devs_title"),
			Content: lang.GetString(langCode, "help_devs_content"),
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_owner": {
			Title:   lang.GetString(langCode, "help_owner_title"),
			Content: lang.GetString(langCode, "help_owner_content"),
			Markup:  core.BackHelpMenuKeyboard(),
		},
	}
}

// helpCallbackHandler handles callbacks from the help keyboard.
// It takes a telegram.CallbackQuery object as input.
// It returns an error if any.
func helpCallbackHandler(cb *telegram.CallbackQuery) error {
	data := cb.DataString()
	chatID, _ := getPeerId(cb.Client, cb.ChatID)
	ctx, cancel := db.Ctx()
	defer cancel()

	langCode := db.Instance.GetLang(ctx, chatID)
	helpCategories := getHelpCategories(langCode)
	if strings.Contains(data, "help_all") {
		_, _ = cb.Answer(lang.GetString(langCode, "opening_help_menu"), &telegram.CallbackOptions{Alert: true})
		response := fmt.Sprintf(lang.GetString(langCode, "start_text"), cb.Sender.FirstName, cb.Client.Me().FirstName)
		_, _ = cb.Edit(response, &telegram.SendOptions{ReplyMarkup: core.HelpMenuKeyboard()})
		return nil
	}

	if strings.Contains(data, "help_back") {
		_, _ = cb.Answer(lang.GetString(langCode, "returning_to_home"), &telegram.CallbackOptions{Alert: true})
		response := fmt.Sprintf(lang.GetString(langCode, "start_text"), cb.Sender.FirstName, cb.Client.Me().FirstName)
		_, _ = cb.Edit(response, &telegram.SendOptions{ReplyMarkup: core.AddMeMarkup(cb.Client.Me().Username)})
		return nil
	}

	if category, ok := helpCategories[data]; ok {
		_, _ = cb.Answer(fmt.Sprintf(lang.GetString(langCode, "opening_category"), category.Title), &telegram.CallbackOptions{Alert: true})
		text := fmt.Sprintf(lang.GetString(langCode, "help_category_text"), category.Title, category.Content)
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: category.Markup})
		return nil
	}

	_, _ = cb.Answer(lang.GetString(langCode, "unknown_command_category"), &telegram.CallbackOptions{Alert: true})
	return nil
}

// privacyHandler handles the /privacy command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func privacyHandler(m *telegram.NewMessage) error {
	chatID, _ := getPeerId(m.Client, m.ChatID())
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, chatID)
	botName := m.Client.Me().FirstName

	text := fmt.Sprintf(lang.GetString(langCode, "privacy_policy"), botName, botName, botName, botName, botName)

	_, err := m.Reply(text, telegram.SendOptions{LinkPreview: false})
	return err
}
