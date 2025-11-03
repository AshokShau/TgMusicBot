/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

import "github.com/AshokShau/TgMusicBot/internal/vc/ntgcalls"

func (ctx *Context) Time(chatId any, streamMode ntgcalls.StreamMode) (uint64, error) {
	parsedChatId, err := ctx.parseChatId(chatId)
	if err != nil {
		return 0, err
	}
	return ctx.binding.Time(parsedChatId, streamMode)
}
