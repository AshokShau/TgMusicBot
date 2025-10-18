package handlers

import (
	"fmt"
	"strconv"
	"tgmusic/pkg/core/cache"

	"github.com/amarnathcjd/gogram/telegram"
)

// loopHandler handles the /loop command.
func loopHandler(m *telegram.NewMessage) error {
	chatId, _ := getPeerId(m.Client, m.ChatID())
	if !cache.ChatCache.IsActive(chatId) {
		_, err := m.Reply("⏸ There is no track currently playing.")
		return err
	}

	args := m.Args()
	if args == "" {
		_, err := m.Reply("<b>🔁 Loop Control</b>\n\n<b>Usage:</b> <code>/loop [count]</code>\n• <code>0</code> to disable loop\n• <code>1-10</code> to set the loop count")
		return err
	}

	argsInt, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.Reply("❌ Invalid loop count provided. Please use a number between 0 and 10.")
		return nil
	}

	if argsInt < 0 || argsInt > 10 {
		_, err = m.Reply("⚠️ The loop count must be between 0 and 10.")
		return err
	}

	cache.ChatCache.SetLoopCount(chatId, argsInt)
	var action string
	if argsInt == 0 {
		action = "Looping has been disabled"
	} else {
		action = fmt.Sprintf("The loop has been set to %d time(s)", argsInt)
	}

	_, err = m.Reply(fmt.Sprintf("🔁 %s.\n\n└ Changed by: %s", action, m.Sender.FirstName))
	return err
}
