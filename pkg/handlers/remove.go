package handlers

import (
	"fmt"
	"strconv"
	"tgmusic/pkg/core/cache"

	"github.com/amarnathcjd/gogram/telegram"
)

// removeHandler handles the /remove command.
func removeHandler(m *telegram.NewMessage) error {
	chatId, _ := getPeerId(m.Client, m.ChatID())

	if !cache.ChatCache.IsActive(chatId) {
		_, _ = m.Reply("⏸ There is no track currently playing.")
		return nil
	}

	queue := cache.ChatCache.GetQueue(chatId)
	if len(queue) == 0 {
		_, _ = m.Reply("📭 The queue is currently empty.")
		return nil
	}

	args := m.Args()
	if args == "" {
		_, _ = m.Reply("<b>❌ Remove Track</b>\n\n<b>Usage:</b> <code>/remove [track number]</code>\n\n- Use <code>1</code> to remove the first track, <code>2</code> for the second, and so on.")
		return nil
	}

	trackNum, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.Reply("⚠️ Please enter a valid track number.")
		return nil
	}

	if trackNum <= 0 || trackNum > len(queue) {
		_, _ = m.Reply(fmt.Sprintf("⚠️ The track number is not valid. Please choose a number between 1 and %d.", len(queue)))
		return nil
	}

	cache.ChatCache.RemoveTrack(chatId, trackNum)
	_, err = m.Reply(fmt.Sprintf("✅ Track #%d has been removed by %s.", trackNum, m.Sender.FirstName))
	return err
}
