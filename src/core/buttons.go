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

	"github.com/amarnathcjd/gogram/telegram"
)

var CloseBtn = telegram.Button.Data("Close", "vcplay_close").Icon(5019500511871632068)

var HomeBtn = telegram.Button.Data("Home", "help_back")

var HelpBtn = telegram.Button.Data("Help", "help_all")

var UserBtn = telegram.Button.Data("Users", "help_user")

var AdminBtn = telegram.Button.Data("Admins", "help_admin")

var OwnerBtn = telegram.Button.Data("Owner", "help_owner")

var DevsBtn = telegram.Button.Data("Devs", "help_devs")

var PlaylistBtn = telegram.Button.Data("Playlist", "help_playlist")

var SourceCodeBtn = telegram.Button.URL("Source Code", "https://github.com/AshokShau/TgMusicBot").Icon(4999005636604723783)

func SupportKeyboard() *telegram.ReplyInlineMarkup {
	channelBtn := telegram.Button.URL("Updates", config.Conf.SupportChannel).Icon(5017470156276761427)
	groupBtn := telegram.Button.URL("Group", config.Conf.SupportGroup).Icon(5944970130554359187)
	keyboard := telegram.NewKeyboard().
		AddRow(channelBtn, groupBtn).
		AddRow(CloseBtn)

	return keyboard.Build()
}

func SettingsKeyboard(playMode, adminMode string) *telegram.ReplyInlineMarkup {
	createButton := func(label, settingType, settingValue, currentValue string) *telegram.KeyboardButtonCallback {
		text := label
		if settingValue == currentValue {
			text += " ✅"
		}
		return telegram.Button.Data(text, fmt.Sprintf("settings_%s_%s", settingType, settingValue))
	}

	keyboard := telegram.NewKeyboard()

	keyboard.AddRow(telegram.Button.Data("🎵 Play Mode", "settings_xxx_noop"))
	keyboard.AddRow(
		createButton("Admins", "play", utils.Admins, playMode),
		createButton("Auth", "play", utils.Auth, playMode),
		createButton("Everyone", "play", utils.Everyone, playMode),
	)

	keyboard.AddRow(telegram.Button.Data("🛡 Admin Mode", "settings_xxx_none"))
	keyboard.AddRow(
		createButton("Admins", "admin", utils.Admins, adminMode),
		createButton("Auth", "admin", utils.Auth, adminMode),
		createButton("Everyone", "admin", utils.Everyone, adminMode),
	)

	keyboard.AddRow(CloseBtn)
	return keyboard.Build()
}

func HelpMenuKeyboard() *telegram.ReplyInlineMarkup {
	keyboard := telegram.NewKeyboard().
		AddRow(UserBtn, AdminBtn, OwnerBtn).
		AddRow(PlaylistBtn, DevsBtn, CloseBtn).
		AddRow(HomeBtn)

	return keyboard.Build()
}

func BackHelpMenuKeyboard() *telegram.ReplyInlineMarkup {
	keyboard := telegram.NewKeyboard().
		AddRow(HelpBtn, HomeBtn).
		AddRow(CloseBtn, SourceCodeBtn)

	return keyboard.Build()
}

func ControlButtons(mode string) *telegram.ReplyInlineMarkup {
	skipBtn := telegram.Button.Data("‣‣I", "play_skip")
	stopBtn := telegram.Button.Data("▢", "play_stop")
	pauseBtn := telegram.Button.Data("II", "play_pause")
	resumeBtn := telegram.Button.Data("▷", "play_resume")
	muteBtn := telegram.Button.Data("🔇", "play_mute")
	unmuteBtn := telegram.Button.Data("🔊", "play_unmute")
	addToPlaylistBtn := telegram.Button.Data("➕", "play_add_to_list")

	var keyboard *telegram.KeyboardBuilder

	switch mode {
	case "play":
		keyboard = telegram.NewKeyboard().AddRow(skipBtn, stopBtn, pauseBtn, resumeBtn).AddRow(addToPlaylistBtn, CloseBtn)
	case "pause":
		keyboard = telegram.NewKeyboard().AddRow(skipBtn, stopBtn, resumeBtn).AddRow(CloseBtn)
	case "resume":
		keyboard = telegram.NewKeyboard().AddRow(skipBtn, stopBtn, pauseBtn).AddRow(CloseBtn)
	case "mute":
		keyboard = telegram.NewKeyboard().AddRow(skipBtn, stopBtn, unmuteBtn).AddRow(CloseBtn)
	case "unmute":
		keyboard = telegram.NewKeyboard().AddRow(skipBtn, stopBtn, muteBtn).AddRow(CloseBtn)
	default:
		keyboard = telegram.NewKeyboard().AddRow(CloseBtn)
	}

	return keyboard.Build()
}

func AddMeMarkup(username string) *telegram.ReplyInlineMarkup {
	addMeBtn := telegram.Button.URL(fmt.Sprintf("Aᴅᴅ ᴍᴇ ᴛᴏ ʏᴏᴜʀ ɢʀᴏᴜᴘ"), fmt.Sprintf("https://t.me/%s?startgroup=true", username)).Icon(5397916757333654639)
	channelBtn := telegram.Button.URL("Updates", config.Conf.SupportChannel).Icon(5017470156276761427)
	groupBtn := telegram.Button.URL("Group", config.Conf.SupportGroup).Icon(5944970130554359187)
	keyboard := telegram.NewKeyboard().
		AddRow(addMeBtn).
		AddRow(HelpBtn).
		AddRow(channelBtn, groupBtn).
		AddRow(SourceCodeBtn)

	return keyboard.Build()
}
