/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

import (
	"fmt"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) getInputGroupCall(chatId int64) (tg.InputGroupCall, error) {
	ctx.inputGroupCallsMutex.RLock()
	if call, ok := ctx.inputGroupCalls[chatId]; ok {
		ctx.inputGroupCallsMutex.RUnlock()
		if call == nil {
			return nil, fmt.Errorf("group call for chatId %d is closed", chatId)
		}
		return call, nil
	}
	ctx.inputGroupCallsMutex.RUnlock()

	peer, err := ctx.App.ResolvePeer(chatId)
	if err != nil {
		return nil, err
	}
	switch chatPeer := peer.(type) {
	case *tg.InputPeerChannel:
		fullChat, err := ctx.App.ChannelsGetFullChannel(
			&tg.InputChannelObj{
				ChannelID:  chatPeer.ChannelID,
				AccessHash: chatPeer.AccessHash,
			},
		)
		if err != nil {
			return nil, err
		}
		ctx.inputGroupCallsMutex.Lock()
		ctx.inputGroupCalls[chatId] = fullChat.FullChat.(*tg.ChannelFull).Call
		ctx.inputGroupCallsMutex.Unlock()
	case *tg.InputPeerChat:
		fullChat, err := ctx.App.MessagesGetFullChat(chatPeer.ChatID)
		if err != nil {
			return nil, err
		}
		ctx.inputGroupCallsMutex.Lock()
		ctx.inputGroupCalls[chatId] = fullChat.FullChat.(*tg.ChatFullObj).Call
		ctx.inputGroupCallsMutex.Unlock()
	default:
		return nil, fmt.Errorf("chatId %d is not a group call", chatId)
	}

	ctx.inputGroupCallsMutex.RLock()
	defer ctx.inputGroupCallsMutex.RUnlock()
	if call, ok := ctx.inputGroupCalls[chatId]; ok && call == nil {
		return nil, fmt.Errorf("group call for chatId %d is closed", chatId)
	} else if ok {
		return call, nil
	}

	return nil, fmt.Errorf("group call for chatId %d not found", chatId)
}
