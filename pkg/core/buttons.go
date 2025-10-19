// Copyright (c) 2025 AshokShau
// Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
// Part of the TgMusicBot project. All rights reserved where applicable.

package core

import (
	"fmt"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"

	"github.com/amarnathcjd/gogram/telegram"
)

// CloseBtn is a button that closes the current view.
var CloseBtn = telegram.Button.Data("Cʟᴏsᴇ", "vcplay_close")

// HomeBtn is a button that returns to the home screen.
var HomeBtn = telegram.Button.Data("Hᴏᴍᴇ", "help_back")

// HelpBtn is a button that displays the help menu.
var HelpBtn = telegram.Button.Data("Hᴇʟᴘ & Cᴏᴍᴍᴀɴᴅꜱ", "help_all")

// UserBtn is a button that displays the user commands.
var UserBtn = telegram.Button.Data("Uꜱᴇʀ Cᴏᴍᴍᴀɴᴅꜱ", "help_user")

// AdminBtn is a button that displays the admin commands.
var AdminBtn = telegram.Button.Data("Aᴅᴍɪɴ Cᴏᴍᴍᴀɴᴅꜱ", "help_admin")

// OwnerBtn is a button that displays the owner commands.
var OwnerBtn = telegram.Button.Data("Oᴡɴᴇʀ Cᴏᴍᴍᴀɴᴅꜱ", "help_owner")

// DevsBtn is a button that displays the developer commands.
var DevsBtn = telegram.Button.Data("Dᴇᴠꜱ Cᴏᴍᴍᴀɴᴅꜱ", "help_devs")

// ChannelBtn is a button that links to the updates channel.
var ChannelBtn = telegram.Button.URL("ᴜᴘᴅᴀᴛᴇꜱ", "https://t.me/FallenProjects")

// GroupBtn is a button that links to the support group.
var GroupBtn = telegram.Button.URL("ꜱᴜᴘᴘᴏʀᴛ", "https://t.me/GuardxSupport")

// SourceCodeBtn is a button that links to the source code.
var SourceCodeBtn = telegram.Button.URL("Sᴏᴜʀᴄᴇ Cᴏᴅᴇ", "https://github.com/AshokShau/TgMusicBot")

// SupportKeyboard creates and returns an inline keyboard with buttons for support and updates.
func SupportKeyboard() *telegram.ReplyInlineMarkup {
	keyboard := telegram.NewKeyboard().
		AddRow(ChannelBtn, GroupBtn).
		AddRow(CloseBtn)

	return keyboard.Build()
}

// SettingsKeyboard creates an inline keyboard for bot settings
func SettingsKeyboard(playMode, adminMode string) *telegram.ReplyInlineMarkup {
	// Helper function to create a button with a checkmark if active
	createButton := func(label, settingType, settingValue, currentValue string) *telegram.KeyboardButtonCallback {
		text := label
		if settingValue == currentValue {
			text += " ✅"
		}
		return telegram.Button.Data(text, fmt.Sprintf("settings_%s_%s", settingType, settingValue))
	}

	keyboard := telegram.NewKeyboard()

	// Play Mode Section
	keyboard.AddRow(telegram.Button.Data("🎵 Play Mode", "settings_xxx_noop"))
	keyboard.AddRow(
		createButton("Admins", "play", cache.Admins, playMode),
		createButton("Auth", "play", cache.Auth, playMode),
		createButton("Everyone", "play", cache.Everyone, playMode),
	)

	// Admin Mode Section
	keyboard.AddRow(telegram.Button.Data("🛡️ Admin Mode", "settings_xxx_none"))
	keyboard.AddRow(
		createButton("Admins", "admin", cache.Admins, adminMode),
		createButton("Auth", "admin", cache.Auth, adminMode),
		createButton("Everyone", "admin", cache.Everyone, adminMode),
	)

	// Close button
	keyboard.AddRow(CloseBtn)

	return keyboard.Build()
}

// HelpMenuKeyboard creates and returns an inline keyboard with buttons for navigating the help menu.
func HelpMenuKeyboard() *telegram.ReplyInlineMarkup {
	keyboard := telegram.NewKeyboard().
		AddRow(UserBtn, AdminBtn).
		AddRow(OwnerBtn, DevsBtn).
		AddRow(CloseBtn, HomeBtn)

	return keyboard.Build()
}

// BackHelpMenuKeyboard creates and returns an inline keyboard with buttons to return to the main help menu.
func BackHelpMenuKeyboard() *telegram.ReplyInlineMarkup {
	keyboard := telegram.NewKeyboard().
		AddRow(HelpBtn, HomeBtn).
		AddRow(CloseBtn, SourceCodeBtn)

	return keyboard.Build()
}

// ControlButtons creates and returns an inline keyboard with playback control buttons, customized based on the current mode.
// The 'mode' parameter can be "play", "pause", "resume", "mute", or "unmute" to display the relevant controls.
func ControlButtons(mode string) *telegram.ReplyInlineMarkup {
	skipBtn := telegram.Button.Data("‣‣I", "play_skip")
	stopBtn := telegram.Button.Data("▢", "play_stop")
	pauseBtn := telegram.Button.Data("II", "play_pause")
	resumeBtn := telegram.Button.Data("▷", "play_resume")
	muteBtn := telegram.Button.Data("🔇", "play_mute")
	unmuteBtn := telegram.Button.Data("🔊", "play_unmute")

	var keyboard *telegram.KeyboardBuilder

	switch mode {
	case "play":
		keyboard = telegram.NewKeyboard().AddRow(skipBtn, stopBtn, pauseBtn, resumeBtn).AddRow(CloseBtn)
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

// AddMeMarkup creates and returns an inline keyboard with a button that allows users to add the bot to their group.
// It requires the bot's username to generate the correct link.
func AddMeMarkup(username string) *telegram.ReplyInlineMarkup {
	addMeBtn := telegram.Button.URL(fmt.Sprintf("Aᴅᴅ ᴍᴇ ᴛᴏ ʏᴏᴜʀ ɢʀᴏᴜᴘ"), fmt.Sprintf("https://t.me/%s?startgroup=true", username))

	keyboard := telegram.NewKeyboard().
		AddRow(addMeBtn).
		AddRow(HelpBtn, SourceCodeBtn).
		AddRow(ChannelBtn, GroupBtn)

	return keyboard.Build()
}
