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
			Title:   "User Commands",
			Content: "<b>Playback:</b>\n• <code>/play [song]</code> — Play a track\n\n<b>Utilities:</b>\n• <code>/start</code> — Start the bot\n• <code>/privacy</code> — View privacy policy\n• <code>/queue</code> — Show current queue",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_admin": {
			Title:   "Admin Commands",
			Content: "<b>Controls:</b>\n• <code>/skip</code> — Skip the current track\n• <code>/pause</code> — Pause playback\n• <code>/resume</code> — Resume playback\n• <code>/seek [sec]</code> — Seek to position\n\n<b>Queue:</b>\n• <code>/remove [x]</code> — Remove a track\n• <code>/loop [0-10]</code> — Set loop count\n\n<b>Access:</b>\n• <code>/auth [reply]</code> — Authorize user\n• <code>/unauth [reply]</code> — Remove authorization\n• <code>/authlist</code> — List authorized users",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_devs": {
			Title:   "Developer Commands",
			Content: "<b>System:</b>\n• <code>/stats</code> — Show usage statistics\n\n<b>Maintenance:</b>\n• <code>/av</code> — Active voice chats",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_owner": {
			Title:   "Owner Commands",
			Content: "<b>Settings:</b>\n• <code>/settings</code> — Chat settings",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_playlist": {
			Title:   "Playlist Commands",
			Content: "<b>Management:</b>\n• <code>/createplaylist [name]</code> — Create a playlist\n• <code>/deleteplaylist [id]</code> — Delete a playlist\n• <code>/addtoplaylist [id] [url]</code> — Add a track\n• <code>/removefromplaylist [id] [url]</code> — Remove a track\n• <code>/playlistinfo [id]</code> — Show playlist info\n• <code>/myplaylists</code> — List your playlists",
			Markup:  core.BackHelpMenuKeyboard(),
		},
	}
}

func helpCallbackHandler(c *td.Client, ctx *td.Context) error {
	cb := ctx.Update.UpdateNewCallbackQuery
	data := cb.DataString()

	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "User", Id: cb.SenderUserId}
	}

	helpCategories := getHelpCategories()

	if strings.Contains(data, "help_all") {
		_ = cb.Answer(c, 0, false, "Opening help menu...", "")
		response := fmt.Sprintf("Hello %s,\n\nI am %s, a fast and powerful music player for Telegram.\n\n<b>Supported platforms:</b> YouTube, Spotify, Apple Music, SoundCloud.\n\nUse the buttons below to explore available commands.", user.FirstName, c.Me.FirstName)
		_, _ = cb.EditMessageText(c, response, &td.EditTextMessageOpts{ReplyMarkup: core.HelpMenuKeyboard(), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	if strings.Contains(data, "help_back") {
		_ = cb.Answer(c, 0, false, "Returning to main menu...", "")
		response := fmt.Sprintf("Hello %s,\n\nI am %s, a fast and powerful music player for Telegram.\n\n<b>Supported platforms:</b> YouTube, Spotify, Apple Music, SoundCloud.\n\nUse the buttons below to explore available commands.", user.FirstName, c.Me.FirstName)
		_, _ = cb.EditMessageText(c, response, &td.EditTextMessageOpts{ReplyMarkup: core.AddMeMarkup(c.Me.Usernames.EditableUsername), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	if category, ok := helpCategories[data]; ok {
		_ = cb.Answer(c, 0, false, category.Title, "")
		text := fmt.Sprintf("<b>%s</b>\n\n%s\n\n<i>Use the buttons below to go back.</i>", category.Title, category.Content)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: category.Markup, ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	_ = cb.Answer(c, 0, true, "Unknown help category.", "")
	return nil
}
