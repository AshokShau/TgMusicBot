/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package core

import (
	"ashokshau/tgmusic/src/config"
	"fmt"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/lang"

	"github.com/amarnathcjd/gogram/telegram"
)

// CloseBtn is a button that closes the current view.
var CloseBtn = telegram.Button.Data("Close", "vcplay_close")

// HomeBtn is a button that returns to the home screen.
var HomeBtn = telegram.Button.Data("Home", "help_back")

// HelpBtn is a button that displays the help menu.
var HelpBtn = telegram.Button.Data("Help", "help_all")

// UserBtn is a button that displays the user commands.
var UserBtn = telegram.Button.Data("User", "help_user")

// AdminBtn is a button that displays the admin commands.
var AdminBtn = telegram.Button.Data("Admin", "help_admin")

// OwnerBtn is a button that displays the owner commands.
var OwnerBtn = telegram.Button.Data("Owner", "help_owner")

// DevsBtn is a button that displays the developer commands.
var DevsBtn = telegram.Button.Data("Devs", "help_devs")

// PlaylistBtn is a button that displays the playlist commands.
var PlaylistBtn = telegram.Button.Data("Playlist", "help_playlist")

// SourceCodeBtn is a button that links to the source code.
var SourceCodeBtn = telegram.Button.URL("Source Code", "")

// SupportKeyboard creates and returns an inline keyboard with buttons for support and updates.
func SupportKeyboard() *telegram.ReplyInlineMarkup {
	channelBtn := telegram.Button.URL("Updates", config.Conf.SupportChannel)
	groupBtn := telegram.Button.URL("Support", config.Conf.SupportGroup)
	keyboard := telegram.NewKeyboard().
		AddRow(channelBtn, groupBtn).
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
		AddRow(PlaylistBtn).
		AddRow(CloseBtn, HomeBtn)

	return keyboard.Build()
}

// BackHelpMenuKeyboard creates and returns an inline keyboard with buttons to return to the main help menu.
func BackHelpMenuKeyboard() *telegram.ReplyInlineMarkup {
	keyboard := telegram.NewKeyboard().
		AddRow(HelpBtn, HomeBtn).
		AddRow(CloseBtn)

	return keyboard.Build()
}

// ControlButtons creates and returns an inline keyboard with playback control buttons, customized based on the current mode.
// The 'mode' parameter can be "play", "pause", "resume", "mute", or "unmute" to display the relevant controls.
func ControlButtons(mode string) *telegram.ReplyInlineMarkup {
	skipBtn := telegram.Button.Data("⏩", "play_skip")
	stopBtn := telegram.Button.Data("⏹️", "play_stop")
	pauseBtn := telegram.Button.Data("⏸️", "play_pause")
	resumeBtn := telegram.Button.Data("▶️", "play_resume")
	muteBtn := telegram.Button.Data("🔇", "play_mute")
	unmuteBtn := telegram.Button.Data("🔊", "play_unmute")
	addToPlaylistBtn := telegram.Button.Data("⭐️ Plist", "play_add_to_list")
	GroupAdBtn := telegram.Button.URL("💬 Group", fmt.Sprintf("https://t.me/+jqbmBXrGuaIyOTRl"))

	var keyboard *telegram.KeyboardBuilder

	switch mode {
	case "play":
		keyboard = telegram.NewKeyboard().AddRow(skipBtn, stopBtn, pauseBtn).AddRow(addToPlaylistBtn, GroupAdBtn).AddRow(CloseBtn)
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

func LanguageKeyboard() *telegram.ReplyInlineMarkup {
	keyboard := telegram.NewKeyboard()
	langs := lang.GetAvailableLangs()
	for i := 0; i < len(langs); i += 2 {
		if i+1 < len(langs) {
			keyboard.AddRow(
				telegram.Button.Data(lang.GetLangDisplayName(langs[i]), fmt.Sprintf("setlang_%s", langs[i])),
				telegram.Button.Data(lang.GetLangDisplayName(langs[i+1]), fmt.Sprintf("setlang_%s", langs[i+1])),
			)
		} else {
			keyboard.AddRow(telegram.Button.Data(lang.GetLangDisplayName(langs[i]), fmt.Sprintf("setlang_%s", langs[i])))
		}
	}
	keyboard.AddRow(CloseBtn)
	return keyboard.Build()
}

// AddMeMarkup creates and returns an inline keyboard with a button that allows users to add the bot to their group.
// It requires the bot's username to generate the correct link.
func AddMeMarkup(username string) *telegram.ReplyInlineMarkup {
	addMeBtn := telegram.Button.URL(fmt.Sprintf("Add me to your group"), fmt.Sprintf("https://t.me/%s?startgroup=true", username))
	channelBtn := telegram.Button.URL("Updates", config.Conf.SupportChannel)
	groupBtn := telegram.Button.URL("Support", config.Conf.SupportGroup)
	ownerBtn := telegram.Button.URL("Owner", fmt.Sprintf("https://t.me/%s", config.Conf.OwnerUsername))
	keyboard := telegram.NewKeyboard().
		AddRow(addMeBtn).
		AddRow(HelpBtn, ownerBtn).
		AddRow(channelBtn, groupBtn)

	return keyboard.Build()
}
