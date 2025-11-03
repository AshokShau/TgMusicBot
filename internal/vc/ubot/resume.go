/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

func (ctx *Context) Resume(chatId any) (bool, error) {
	parsedChatId, err := ctx.parseChatId(chatId)
	if err != nil {
		return false, err
	}
	return ctx.binding.Resume(parsedChatId)
}
