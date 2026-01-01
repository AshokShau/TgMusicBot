/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

import (
	"ashokshau/tgmusic/src/vc/ntgcalls"
)

func (ctx *Context) Play(chatId int64, mediaDescription ntgcalls.MediaDescription) error {
	if ctx.binding.Calls()[chatId] != nil {
		return ctx.binding.SetStreamSources(chatId, ntgcalls.CaptureStream, mediaDescription)
	}
	err := ctx.connectCall(chatId, mediaDescription, "")
	if err != nil {
		return err
	}
	if chatId < 0 {
		err = ctx.joinPresentation(chatId, mediaDescription.Screen != nil)
		if err != nil {
			return err
		}
		return ctx.updateSources(chatId)
	}
	return nil
}
