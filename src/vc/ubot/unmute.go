/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

func (ctx *Context) Unmute(chatId int64) (bool, error) {
	return ctx.binding.UnMute(chatId)
}
