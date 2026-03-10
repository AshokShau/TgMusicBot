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

func cb(text, data string) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeCallback{
			Data: []byte(data),
		},
	}
}

func url(text, link string) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeUrl{
			Url: link,
		},
	}
}

var CloseBtn = cb("Close", "vcplay_close")
var HomeBtn = cb("Home", "help_back")
var HelpBtn = cb("Help", "help_all")
var UserBtn = cb("Users", "help_user")
var AdminBtn = cb("Admins", "help_admin")
var OwnerBtn = cb("Owner", "help_owner")
var DevsBtn = cb("Devs", "help_devs")
var PlaylistBtn = cb("Playlist", "help_playlist")

var SourceCodeBtn = url("Source Code", "https://github.com/AshokShau/TgMusicBot")

func SupportKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {

	channelBtn := url("Updates", config.Conf.SupportChannel)
	groupBtn := url("Group", config.Conf.SupportGroup)

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
			{CloseBtn},
		},
	}
}

func SettingsKeyboard(playMode, adminMode string) *gotdbot.ReplyMarkupInlineKeyboard {

	createButton := func(label, settingType, settingValue, currentValue string) gotdbot.InlineKeyboardButton {

		text := label
		if settingValue == currentValue {
			text += " ✅"
		}

		return cb(text, fmt.Sprintf("settings_%s_%s", settingType, settingValue))
	}

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{

			{cb("🎵 Play Mode", "settings_xxx_noop")},

			{
				createButton("Admins", "play", utils.Admins, playMode),
				createButton("Auth", "play", utils.Auth, playMode),
				createButton("Everyone", "play", utils.Everyone, playMode),
			},

			{cb("🛡️ Admin Mode", "settings_xxx_none")},

			{
				createButton("Admins", "admin", utils.Admins, adminMode),
				createButton("Auth", "admin", utils.Auth, adminMode),
				createButton("Everyone", "admin", utils.Everyone, adminMode),
			},

			{CloseBtn},
		},
	}
}

func HelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{UserBtn, AdminBtn, OwnerBtn},
			{PlaylistBtn, DevsBtn, CloseBtn},
			{HomeBtn},
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

	skipBtn := cb("‣‣I", "play_skip")
	stopBtn := cb("▢", "play_stop")
	pauseBtn := cb("II", "play_pause")
	resumeBtn := cb("▷", "play_resume")
	muteBtn := cb("🔇", "play_mute")
	unmuteBtn := cb("🔊", "play_unmute")
	addToPlaylistBtn := cb("➕", "play_add_to_list")

	switch mode {

	case "play":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, pauseBtn, resumeBtn},
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
	)

	channelBtn := url("Updates", config.Conf.SupportChannel)
	groupBtn := url("Group", config.Conf.SupportGroup)

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{addMeBtn},
			{HelpBtn},
			{channelBtn, groupBtn},
			{SourceCodeBtn},
		},
	}
}
