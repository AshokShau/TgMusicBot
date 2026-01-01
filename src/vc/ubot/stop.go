/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

func (ctx *Context) Stop(chatId int64) error {
	ctx.presentations = stdRemove(ctx.presentations, chatId)
	delete(ctx.callSources, chatId)
	err := ctx.binding.Stop(chatId)
	if err != nil {
		return err
	}
	ctx.inputGroupCallsMutex.RLock()
	inputGroupCall := ctx.inputGroupCalls[chatId]
	ctx.inputGroupCallsMutex.RUnlock()
	_, err = ctx.App.PhoneLeaveGroupCall(inputGroupCall, 0)
	if err != nil {
		return err
	}
	return nil
}
