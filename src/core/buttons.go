/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package core

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/utils"
	"fmt"

	"github.com/AshokShau/gotdbot"
)

func cb(text, data string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeCallback{
			Data: []byte(data),
		},
		Style: style,
	}
}

func userId(text string, userId int64, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text:  text,
		Type:  &gotdbot.InlineKeyboardButtonTypeUser{UserId: userId},
		Style: style,
	}
}

func url(text, link string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeUrl{
			Url: link,
		},
		Style: style,
	}
}

var CloseBtn = cb("Close", "vcplay_close", gotdbot.ButtonStyleDanger{})
var HomeBtn = cb("Home", "help_back", gotdbot.ButtonStylePrimary{})
var HelpBtn = cb("Help", "help_all", gotdbot.ButtonStyleDefault{})
var UserBtn = cb("Users", "help_user", gotdbot.ButtonStyleDefault{})
var AdminBtn = cb("Admins", "help_admin", gotdbot.ButtonStyleDefault{})
var OwnerBtn = cb("Owner", "help_owner", gotdbot.ButtonStyleDefault{})
var DevsBtn = cb("Devs", "help_devs", gotdbot.ButtonStyleDefault{})
var PlaylistBtn = cb("Playlist", "help_playlist", gotdbot.ButtonStyleDefault{})
var AutoplayBtn = cb("Autoplay", "help_autoplay", gotdbot.ButtonStyleDefault{})

var SourceCodeBtn = url("Source Code", "https://github.com/AshokShau/TgMusicBot", gotdbot.ButtonStylePrimary{})
var channelBtn = url("Updates", config.SupportChannel, gotdbot.ButtonStyleDefault{})
var groupBtn = url("Group", config.SupportGroup, gotdbot.ButtonStyleDefault{})

func SupportKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
			{CloseBtn},
		},
	}
}

func SupportBtn() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
		},
	}
}

func SettingsKeyboard(playMode, adminMode string, cmdDelete bool, language string) *gotdbot.ReplyMarkupInlineKeyboard {
	playText := "Everyone"
	if playMode == utils.Admins {
		playText = "Admins"
	}

	deleteText := "False"
	if cmdDelete {
		deleteText = "True"
	}

	adminText := "Everyone"
	if adminMode == utils.Admins {
		adminText = "Admins"
	}

	langText := "English"
	if language != "en" && language != "" {
		langText = language
	}

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{
				cb("Play Mode ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(playText, "settings_play", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb("Command Delete ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(deleteText, "settings_delete", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb("Admin Mode ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(adminText, "settings_admin", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb("Language ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(langText, "settings_lang", gotdbot.ButtonStyleDefault{}),
			},
			{CloseBtn},
		},
	}
}

func HelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{UserBtn, AdminBtn, OwnerBtn},
			{PlaylistBtn, DevsBtn, AutoplayBtn},
			{HomeBtn, CloseBtn},
		},
	}
}

func BackHelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{HelpBtn, HomeBtn},
			{CloseBtn, SourceCodeBtn},
		},
	}
}

func ControlButtons(mode string) *gotdbot.ReplyMarkupInlineKeyboard {
	skipBtn := cb("‣‣I", "play_skip", gotdbot.ButtonStyleDefault{})
	stopBtn := cb("▢", "play_stop", gotdbot.ButtonStyleDefault{})
	pauseBtn := cb("II", "play_pause", gotdbot.ButtonStyleDefault{})
	resumeBtn := cb("▷", "play_resume", gotdbot.ButtonStyleDefault{})
	muteBtn := cb("🔇", "play_mute", gotdbot.ButtonStyleDefault{})
	unmuteBtn := cb("🔊", "play_unmute", gotdbot.ButtonStyleDefault{})
	addToPlaylistBtn := cb("➕", "play_add_to_list", gotdbot.ButtonStylePrimary{})

	switch mode {

	case "play":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, pauseBtn},
				{addToPlaylistBtn, CloseBtn},
			},
		}

	case "pause":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, resumeBtn},
				{CloseBtn},
			},
		}

	case "resume":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, pauseBtn},
				{CloseBtn},
			},
		}

	case "mute":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, unmuteBtn},
				{CloseBtn},
			},
		}

	case "unmute":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, muteBtn},
				{CloseBtn},
			},
		}

	default:
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{CloseBtn},
			},
		}
	}
}

func AddMeMarkup(username string) *gotdbot.ReplyMarkupInlineKeyboard {

	addMeBtn := url(
		"Aᴅᴅ ᴍᴇ ᴛᴏ ʏᴏᴜʀ ɢʀᴏᴜᴘ",
		fmt.Sprintf("https://t.me/%s?startgroup=true", username),
		gotdbot.ButtonStylePrimary{},
	)

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{addMeBtn},
			{HelpBtn},
			{channelBtn, groupBtn},
			{SourceCodeBtn},
		},
	}
}

func PlayNowButton(trackID string) gotdbot.InlineKeyboardButton {
	return cb("Play Now", fmt.Sprintf("play_now_%s", trackID), gotdbot.ButtonStyleDanger{})
}

func QueueMarkup(trackID string) *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{PlayNowButton(trackID), CloseBtn},
		},
	}
}
