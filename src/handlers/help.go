/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"fmt"
	"strings"

	"ashokshau/tgmusic/src/core"

	td "github.com/AshokShau/gotdbot"
)

func getHelpCategories() map[string]struct {
	Title   string
	Content string
	Markup  *td.ReplyMarkupInlineKeyboard
} {
	return map[string]struct {
		Title   string
		Content string
		Markup  *td.ReplyMarkupInlineKeyboard
	}{
		"help_user": {
			Title:   "🎧 User Commands",
			Content: "<b>Playback:</b>\n• <code>/play [song]</code> — Play music\n\n<b>Utilities:</b>\n• <code>/start</code> — Start bot\n• <code>/privacy</code> — Privacy Policy\n• <code>/queue</code> — View queue",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_admin": {
			Title:   "⚙️ Admin Commands",
			Content: "<b>Controls:</b>\n• <code>/skip</code> — Skip track\n• <code>/pause</code> — Pause\n• <code>/resume</code> — Resume\n• <code>/seek [sec]</code> — Seek\n\n<b>Queue:</b>\n• <code>/remove [x]</code> — Remove track\n• <code>/loop [0-10]</code> — Loop queue\n\n<b>Access:</b>\n• <code>/auth [reply]</code> — Authorize user\n• <code>/unauth [reply]</code> — Unauthorize\n• <code>/authlist</code> — List authorized",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_devs": {
			Title:   "🛠 Developer Tools",
			Content: "<b>System:</b>\n• <code>/stats</code> — Usage stats\n\n<b>Maintenance:</b>\n• <code>/av</code> — Active voice chats",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_owner": {
			Title:   "🔐 Owner Commands",
			Content: "<b>Settings:</b>\n• <code>/settings</code> — Chat settings",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_playlist": {
			Title:   "🎵 Playlist Commands",
			Content: "<b>Playlist Management:</b>\n• <code>/createplaylist [name]</code> — Create playlist\n• <code>/deleteplaylist [id]</code> — Delete playlist\n• <code>/addtoplaylist [id] [url]</code> — Add song\n• <code>/removefromplaylist [id] [url]</code> — Remove song\n• <code>/playlistinfo [id]</code> — Playlist info\n• <code>/myplaylists</code> — My playlists",
			Markup:  core.BackHelpMenuKeyboard(),
		},
	}
}

// helpCallbackHandler handles callbacks from the help keyboard.
// It returns an error if any.
func helpCallbackHandler(c *td.Client, ctx *td.Context) error {
	cb := ctx.Update.UpdateNewCallbackQuery
	data := cb.DataString()
	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "User", Id: cb.SenderUserId}
	}

	helpCategories := getHelpCategories()
	if strings.Contains(data, "help_all") {
		_ = cb.Answer(c, 300, false, "📖 Displaying all help categories...", "")
		response := fmt.Sprintf("Hello %s!\n\nI am %s, a fast and powerful music player for Telegram.\n\n<b>Supported Platforms:</b> YouTube, Spotify, Apple Music, SoundCloud.\n\nClick the <b>Help</b> button below for more information.", user.FirstName, c.Me().FirstName)
		_, _ = cb.EditMessageText(c, response, &td.EditTextMessageOpts{ReplyMarkup: core.HelpMenuKeyboard(), ParseMode: "HTML"})
		return nil
	}

	if strings.Contains(data, "help_back") {
		_ = cb.Answer(c, 300, false, "🏠 Returning to home...", "")
		response := fmt.Sprintf("Hello %s!\n\nI am %s, a fast and powerful music player for Telegram.\n\n<b>Supported Platforms:</b> YouTube, Spotify, Apple Music, SoundCloud.\n\nClick the <b>Help</b> button below for more information.", user.FirstName, c.Me().FirstName)
		_, _ = cb.EditMessageText(c, response, &td.EditTextMessageOpts{ReplyMarkup: core.AddMeMarkup(c.Me().Usernames.EditableUsername), ParseMode: "HTML"})
		return nil
	}

	if category, ok := helpCategories[data]; ok {
		_ = cb.Answer(c, 300, false, category.Title, "")
		text := fmt.Sprintf("<b>%s</b>\n\n%s\n\n🔙 <i>Use buttons below to go back.</i>", category.Title, category.Content)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: category.Markup, ParseMode: "HTML"})
		return nil
	}

	_ = cb.Answer(c, 300, true, "Unknown help category.", "")
	return nil
}
