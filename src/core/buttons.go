/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
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

var CloseBtn = telegram.Button.Data("❌ ᴄʟᴏꜱᴇ", "vcplay_close")

var HomeBtn = telegram.Button.Data("🏠 ʜᴏᴍᴇ", "help_back")

var HelpBtn = telegram.Button.Data("📖 ʜᴇʟᴘ & ᴄᴏᴍᴍᴀɴᴅꜱ", "help_all")

var UserBtn = telegram.Button.Data("👤 ᴜꜱᴇʀꜱ", "help_user")

var AdminBtn = telegram.Button.Data("🛡 ᴀᴅᴍɪɴꜱ", "help_admin")

var OwnerBtn = telegram.Button.Data("👑 ᴏᴡɴᴇʀ", "help_owner")

var DevsBtn = telegram.Button.Data("‍💻 ᴅᴇᴠꜱ", "help_devs")

var PlaylistBtn = telegram.Button.Data("🎶 ᴘʟᴀʏʟɪꜱᴛ", "help_playlist")

var SourceCodeBtn = telegram.Button.URL("💻 ꜱᴏᴜʀᴄᴇ", "https://github.com/AshokShau/TgMusicBot")

func SupportKeyboard() *telegram.ReplyInlineMarkup {
	channelBtn := telegram.Button.URL("ᴜᴘᴅᴀᴛᴇꜱ", config.Conf.SupportChannel)
	groupBtn := telegram.Button.URL("ꜱᴜᴘᴘᴏʀᴛ", config.Conf.SupportGroup)
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

	keyboard.AddRow(telegram.Button.Data("🛡️ Admin Mode", "settings_xxx_none"))
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
		AddRow(UserBtn, AdminBtn).
		AddRow(OwnerBtn, DevsBtn).
		AddRow(PlaylistBtn).
		AddRow(CloseBtn, HomeBtn)

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
	addToPlaylistBtn := telegram.Button.Data("➕ Playlist", "play_add_to_list")

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
	addMeBtn := telegram.Button.URL(fmt.Sprintf("Aᴅᴅ ᴍᴇ ᴛᴏ ʏᴏᴜʀ ɢʀᴏᴜᴘ"), fmt.Sprintf("https://t.me/%s?startgroup=true", username))
	channelBtn := telegram.Button.URL("ᴜᴘᴅᴀᴛᴇꜱ", config.Conf.SupportChannel)
	groupBtn := telegram.Button.URL("ꜱᴜᴘᴘᴏʀᴛ", config.Conf.SupportGroup)
	keyboard := telegram.NewKeyboard().
		AddRow(addMeBtn).
		AddRow(HelpBtn, SourceCodeBtn).
		AddRow(channelBtn, groupBtn)

	return keyboard.Build()
}
