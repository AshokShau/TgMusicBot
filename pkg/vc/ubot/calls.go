/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

import (
	"github.com/AshokShau/TgMusicBot/pkg/vc/ntgcalls"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) Calls() map[int64]*ntgcalls.CallInfo {
	return ctx.binding.Calls()
}

func (ctx *Context) InputGroupCall(chatId int64) tg.InputGroupCall {
	return ctx.inputGroupCalls[chatId]
}
