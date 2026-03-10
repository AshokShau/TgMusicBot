/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"fmt"
	"time"

	td "github.com/AshokShau/gotdbot"
)

func handleVoiceChatMessage(c *td.Client, ctx *td.Context) error {
	update := ctx.Update.UpdateNewMessage
	m := update.Message
	chatID := ctx.EffectiveChatId

	// Chat is not a Supergroup
	if m.IsGroup() {
		text := fmt.Sprintf(
			"This chat (%d) is not a supergroup yet.\n<b>⚠️ Please convert this chat to a supergroup and add me as admin.</b>\n\nIf you don't know how to convert, use this guide:\n🔗 https://te.legra.ph/How-to-Convert-a-Group-to-a-Supergroup-01-02\n\nIf you have any questions, join our support group:",
			chatID,
		)

		_, _ = c.SendTextMessage(chatID, text, &td.SendTextMessageOpts{
			ReplyMarkup:           core.AddMeMarkup(c.Me().Usernames.EditableUsername),
			DisableWebPagePreview: true,
			ParseMode:             "HTML",
		})

		time.Sleep(1 * time.Second)
		_ = c.LeaveChat(chatID)
		return nil
	}

	if m.Content == nil {
		return nil
	}
	var message string
	switch m.Content.(type) {
	case *td.MessageVideoChatStarted:
		cache.ChatCache.ClearChat(chatID)
		message = "🎙️ Video chat started!\nUse /play <song name> to play music."
	case *td.MessageVideoChatEnded:
		cache.ChatCache.ClearChat(chatID)
		message = "🎧 Video chat ended!\nAll queues cleared."
	default:
		return nil
	}

	_, _ = c.SendTextMessage(chatID, message, nil)
	return td.EndGroups
}
