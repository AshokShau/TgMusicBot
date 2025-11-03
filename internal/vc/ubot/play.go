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

func (ctx *Context) Play(chatId any, mediaDescription ntgcalls.MediaDescription) error {
	parsedChatId, err := ctx.parseChatId(chatId)
	if err != nil {
		return err
	}
	if ctx.binding.Calls()[parsedChatId] != nil {
		return ctx.binding.SetStreamSources(parsedChatId, ntgcalls.CaptureStream, mediaDescription)
	}
	err = ctx.connectCall(parsedChatId, mediaDescription, "")
	if err != nil {
		return err
	}
	if parsedChatId < 0 {
		err = ctx.joinPresentation(parsedChatId, mediaDescription.Screen != nil)
		if err != nil {
			return err
		}
		return ctx.updateSources(parsedChatId)
	}
	return nil
}
