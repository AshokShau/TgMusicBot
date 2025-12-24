/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"fmt"
	"strings"

	"ashokshau/tgmusic/src/core"

	"github.com/amarnathcjd/gogram/telegram"
)

func getHelpCategories() map[string]struct {
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
			Title:   "🎧 User Commands",
			Content: "<b>▶️ Playback:</b>\n• <code>/play [song]</code> — Play audio in VC\n\n<b>🛠 Utilities:</b>\n• <code>/start</code> — Intro message\n• <code>/privacy</code> — Privacy policy\n• <code>/queue</code> — View track queue",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_admin": {
			Title:   "⚙️ Admin Commands",
			Content: "<b>🎛 Playback Controls:</b>\n• <code>/skip</code> — Skip current track\n• <code>/pause</code> — Pause playback\n• <code>/resume</code> — Resume playback\n• <code>/seek [sec]</code> — Jump to a position\n\n<b>📋 Queue Management:</b>\n• <code>/remove [x]</code> — Remove track number x\n• <code>/loop [0-10]</code> — Repeat queue x times\n\n<b>👑 Permissions:</b>\n• <code>/auth [reply]</code> — Grant approval\n• <code>/unauth [reply]</code> — Revoke authorization\n• <code>/authlist</code> — View authorized users",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_devs": {
			Title:   "🛠 Developer Tools",
			Content: "<b>📊 System Tools:</b>\n• <code>/stats</code> — Show usage stats\n\n<b>🧹 Maintenance:</b>\n• <code>/av</code> — Show active voice chats",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_owner": {
			Title:   "🔐 Owner Commands",
			Content: "<b>⚙️ Settings:</b>\n• <code>/settings</code> - Update chat settings",
			Markup:  core.BackHelpMenuKeyboard(),
		},
		"help_playlist": {
			Title:   "🎵 Playlist Commands",
			Content: "<b>🎵 Playlist Management:</b>\n• <code>/createplaylist [name]</code> — Create a new playlist\n• <code>/deleteplaylist [id]</code> — Delete a playlist\n• <code>/addtoplaylist [id] [url]</code> — Add a song to a playlist\n• <code>/removefromplaylist [id] [url]</code> — Remove a song from a playlist\n• <code>/playlistinfo [id]</code> — View playlist details\n• <code>/myplaylists</code> — View your playlists",
			Markup:  core.BackHelpMenuKeyboard(),
		},
	}
}

// helpCallbackHandler handles callbacks from the help keyboard.
// It takes a telegram.CallbackQuery object as input.
// It returns an error if any.
func helpCallbackHandler(cb *telegram.CallbackQuery) error {
	data := cb.DataString()

	helpCategories := getHelpCategories()
	if strings.Contains(data, "help_all") {
		_, _ = cb.Answer("📚 Opening Help Menu...", &telegram.CallbackOptions{Alert: false})
		response := fmt.Sprintf("ʜᴇʏ %s;\n\n◎ ᴛʜɪꜱ ɪꜱ %s!\n➻ ᴀ ꜰᴀꜱᴛ & ᴘᴏᴡᴇʀꜰᴜʟ ᴛᴇʟᴇɢʀᴀᴍ ᴍᴜꜱɪᴄ ᴘʟᴀʏᴇʀ ʙᴏᴛ.\n\nꜱᴜᴘᴘᴏʀᴛᴇᴅ ᴘʟᴀᴛꜰᴏʀᴍꜱ: ʏᴏᴜᴛᴜʙᴇ, ꜱᴘᴏᴛɪꜰʏ, ᴀᴘᴘʟᴇ ᴍᴜꜱɪᴄ, ꜱᴏᴜɴᴅᴄʟᴏᴜᴅ.\n\n---\n◎ ᴄʟɪᴄᴋ ᴏɴ ʜᴇʟᴘ ʙᴜᴛᴛᴏɴ ꜰᴏʀ ɪɴꜰᴏ.", cb.Sender.FirstName, cb.Client.Me().FirstName)
		_, _ = cb.Edit(response, &telegram.SendOptions{ReplyMarkup: core.HelpMenuKeyboard()})
		return nil
	}

	if strings.Contains(data, "help_back") {
		_, _ = cb.Answer("🏠 Returning to home...", &telegram.CallbackOptions{Alert: false})
		response := fmt.Sprintf("ʜᴇʏ %s;\n\n◎ ᴛʜɪꜱ ɪꜱ %s!\n➻ ᴀ ꜰᴀꜱᴛ & ᴘᴏᴡᴇʀꜰᴜʟ ᴛᴇʟᴇɢʀᴀᴍ ᴍᴜꜱɪᴄ ᴘʟᴀʏᴇʀ ʙᴏᴛ.\n\nꜱᴜᴘᴘᴏʀᴛᴇᴅ ᴘʟᴀᴛꜰᴏʀᴍꜱ: ʏᴏᴜᴛᴜʙᴇ, ꜱᴘᴏᴛɪꜰʏ, ᴀᴘᴘʟᴇ ᴍᴜꜱɪᴄ, ꜱᴏᴜɴᴅᴄʟᴏᴜᴅ.\n\n---\n◎ ᴄʟɪᴄᴋ ᴏɴ ʜᴇʟᴘ ʙᴜᴛᴛᴏɴ ꜰᴏʀ ɪɴꜰᴏ.", cb.Sender.FirstName, cb.Client.Me().FirstName)
		_, _ = cb.Edit(response, &telegram.SendOptions{ReplyMarkup: core.AddMeMarkup(cb.Client.Me().Username)})
		return nil
	}

	if category, ok := helpCategories[data]; ok {
		_, _ = cb.Answer(fmt.Sprintf("📖 %s", category.Title), &telegram.CallbackOptions{Alert: false})
		text := fmt.Sprintf("<b>%s</b>\n\n%s\n\n🔙 <i>Use buttons below to go back.</i>", category.Title, category.Content)
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: category.Markup})
		return nil
	}

	_, _ = cb.Answer("⚠️ Unknown command category.", &telegram.CallbackOptions{Alert: false})
	return nil
}

// privacyHandler handles the /privacy command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func privacyHandler(m *telegram.NewMessage) error {
	botName := m.Client.Me().FirstName

	text := fmt.Sprintf("<u><b>Privacy Policy for %s:</b></u>\n\n<b>1. Data Storage:</b>\n- %s does not store any personal data on the user's device.\n- We do not collect or store any data about your device or personal browsing activity.\n\n<b>2. What We Collect:</b>\n- We only collect your Telegram <b>user ID</b> and <b>chat ID</b> to provide the music streaming and interaction functionalities of the bot.\n- No personal data such as your name, phone number, or location is collected.\n\n<b>3. Data Usage:</b>\n- The collected data (Telegram UserID, ChatID) is used strictly to provide the music streaming and interaction functionalities of the bot.\n- We do not use this data for any marketing or commercial purposes.\n\n<b>4. Data Sharing:</b>\n- We do not share any of your personal or chat data with any third parties, organizations, or individuals.\n- No sensitive data is sold, rented, or traded to any outside entities.\n\n<b>5. Data Security:</b>\n- We take reasonable security measures to protect the data we collect. This includes standard practices like encryption and safe storage.\n- However, we cannot guarantee the absolute security of your data, as no online service is 100%% secure.\n\n<b>6. Cookies and Tracking:</b>\n- %s does not use cookies or similar tracking technologies to collect personal information or track your behavior.\n\n<b>7. Third-Party Services:</b>\n- %s does not integrate with any third-party services that collect or process your personal information, aside from Telegram's own infrastructure.\n\n<b>8. Your Rights:</b>\n- You have the right to request the deletion of your data. Since we only store your Telegram ID and chat ID temporarily to function properly, these can be removed upon request.\n- You may also revoke access to the bot at any time by removing or blocking it from your chats.\n\n<b>9. Changes to the Privacy Policy:</b>\n- We may update this privacy policy from time to time. Any changes will be communicated through updates within the bot.\n\n<b>10. Contact Us:</b>\nIf you have any questions or concerns about our privacy policy, feel free to contact us at <a href=\"https://t.me/GuardxSupport\">Support Group</a>\n\n──────────────────\n<b>Note:</b> This privacy policy is in place to help you understand how your data is handled and to ensure that your experience with %s is safe and respectful.", botName, botName, botName, botName, botName)

	_, err := m.Reply(text, &telegram.SendOptions{LinkPreview: false})
	return err
}
