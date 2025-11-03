/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

import (
	"github.com/AshokShau/TgMusicBot/internal/vc/ntgcalls"
)

func (ctx *Context) Record(chatId any, mediaDescription ntgcalls.MediaDescription) error {
	parsedChatId, err := ctx.parseChatId(chatId)
	if err != nil {
		return err
	}
	if ctx.binding.Calls()[parsedChatId] == nil {
		err = ctx.Play(chatId, ntgcalls.MediaDescription{})
		if err != nil {
			return err
		}
	}
	return ctx.binding.SetStreamSources(parsedChatId, ntgcalls.PlaybackStream, mediaDescription)
}
