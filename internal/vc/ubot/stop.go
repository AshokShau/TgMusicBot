/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

func (ctx *Context) Stop(chatId any) error {
	parsedChatId, err := ctx.parseChatId(chatId)
	if err != nil {
		return err
	}
	ctx.presentations = stdRemove(ctx.presentations, parsedChatId)
	delete(ctx.pendingPresentation, parsedChatId)
	delete(ctx.callSources, parsedChatId)
	err = ctx.binding.Stop(parsedChatId)
	if err != nil {
		return err
	}
	_, err = ctx.App.PhoneLeaveGroupCall(ctx.inputGroupCalls[parsedChatId], 0)
	if err != nil {
		return err
	}
	return nil
}
