package handlers

import (
	"fmt"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core"

	"github.com/amarnathcjd/gogram/telegram"
)

var helpCategories = map[string]struct {
	Title   string
	Content string
	Markup  *telegram.ReplyInlineMarkup
}{
	"help_user": {
		Title: "🎧 User Commands",
		Content: `
<b>▶️ Playback:</b>
• <code>/play [song]</code> — Play audio in VC

<b>🛠 Utilities:</b>
• <code>/start</code> — Intro message
• <code>/privacy</code> — Privacy policy
• <code>/queue</code> — View track queue
`,
		Markup: core.BackHelpMenuKeyboard(),
	},
	"help_admin": {
		Title: "⚙️ Admin Commands",
		Content: `
<b>🎛 Playback Controls:</b>
• <code>/skip</code> — Skip current track
• <code>/pause</code> — Pause playback
• <code>/resume</code> — Resume playback
• <code>/seek [sec]</code> — Jump to a position

<b>📋 Queue Management:</b>
• <code>/remove [x]</code> — Remove track number x
• <code>/loop [0-10]</code> — Repeat queue x times

<b>👑 Permissions:</b>
• <code>/auth [reply]</code> — Grant approval
• <code>/unauth [reply]</code> — Revoke authorization
• <code>/authlist</code> — View authorized users
`,
		Markup: core.BackHelpMenuKeyboard(),
	},
	"help_devs": {
		Title: "🛠 Developer Tools",
		Content: `
<b>📊 System Tools:</b>
• <code>/stats</code> — Show usage stats

<b>🧹 Maintenance:</b>
• <code>/av</code> — Show active voice chats
`,
		Markup: core.BackHelpMenuKeyboard(),
	},
	"help_owner": {
		Title: "🔐 Owner Commands",
		Content: `
<b>⚙️ Settings:</b>
• <code>/settings</code> - Update chat settings
`,
		Markup: core.BackHelpMenuKeyboard(),
	},
}

// helpCallbackHandler handles callbacks from the help keyboard.
// It takes a telegram.CallbackQuery object as input.
// It returns an error if any.
func helpCallbackHandler(cb *telegram.CallbackQuery) error {
	data := cb.DataString()
	if strings.Contains(data, "help_all") {
		_, _ = cb.Answer("📚 Opening Help Menu...")
		response := fmt.Sprintf(startText, cb.Sender.FirstName, cb.Client.Me().FirstName)
		_, _ = cb.Edit(response, &telegram.SendOptions{ReplyMarkup: core.HelpMenuKeyboard()})
		return nil
	}

	if strings.Contains(data, "help_back") {
		_, _ = cb.Answer("🏠 Returning to home...")
		response := fmt.Sprintf(startText, cb.Sender.FirstName, cb.Client.Me().FirstName)
		_, _ = cb.Edit(response, &telegram.SendOptions{ReplyMarkup: core.AddMeMarkup(cb.Client.Me().Username)})
		return nil
	}

	if category, ok := helpCategories[data]; ok {
		_, _ = cb.Answer(fmt.Sprintf("📖 %s", category.Title))
		text := fmt.Sprintf("<b>%s</b>\n\n%s\n\n🔙 <i>Use buttons below to go back.</i>", category.Title, category.Content)
		_, _ = cb.Edit(text, &telegram.SendOptions{ReplyMarkup: category.Markup})
		return nil
	}

	_, _ = cb.Answer("⚠️ Unknown command category.", &telegram.CallbackOptions{Alert: true})
	return nil
}

// privacyHandler handles the /privacy command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func privacyHandler(m *telegram.NewMessage) error {
	botName := m.Client.Me().FirstName

	text := fmt.Sprintf(`
<u><b>Privacy Policy for %s:</b></u>

<b>1. Data Storage:</b>
- %s does not store any personal data on the user's device.
- We do not collect or store any data about your device or personal browsing activity.

<b>2. What We Collect:</b>
- We only collect your Telegram <b>user ID</b> and <b>chat ID</b> to provide the music streaming and interaction functionalities of the bot.
- No personal data such as your name, phone number, or location is collected.

<b>3. Data Usage:</b>
- The collected data (Telegram UserID, ChatID) is used strictly to provide the music streaming and interaction functionalities of the bot.
- We do not use this data for any marketing or commercial purposes.

<b>4. Data Sharing:</b>
- We do not share any of your personal or chat data with any third parties, organizations, or individuals.
- No sensitive data is sold, rented, or traded to any outside entities.

<b>5. Data Security:</b>
- We take reasonable security measures to protect the data we collect. This includes standard practices like encryption and safe storage.
- However, we cannot guarantee the absolute security of your data, as no online service is 100%% secure.

<b>6. Cookies and Tracking:</b>
- %s does not use cookies or similar tracking technologies to collect personal information or track your behavior.

<b>7. Third-Party Services:</b>
- %s does not integrate with any third-party services that collect or process your personal information, aside from Telegram's own infrastructure.

<b>8. Your Rights:</b>
- You have the right to request the deletion of your data. Since we only store your Telegram ID and chat ID temporarily to function properly, these can be removed upon request.
- You may also revoke access to the bot at any time by removing or blocking it from your chats.

<b>9. Changes to the Privacy Policy:</b>
- We may update this privacy policy from time to time. Any changes will be communicated through updates within the bot.

<b>10. Contact Us:</b>
If you have any questions or concerns about our privacy policy, feel free to contact us at <a href="https://t.me/GuardxSupport">Support Group</a>

──────────────────
<b>Note:</b> This privacy policy is in place to help you understand how your data is handled and to ensure that your experience with %s is safe and respectful.
`, botName, botName, botName, botName, botName)

	_, err := m.Reply(text, telegram.SendOptions{LinkPreview: false})
	return err
}
