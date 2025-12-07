package ubot

import (
	"fmt"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) getInputGroupCall(chatId int64) (tg.InputGroupCall, error) {
	ctx.groupCallsMutex.RLock()
	call, ok := ctx.inputGroupCalls[chatId]
	ctx.groupCallsMutex.RUnlock()

	if ok {
		if call == nil {
			return nil, fmt.Errorf("group call for chatId %d is closed", chatId)
		}
		return call, nil
	}

	peer, err := ctx.App.ResolvePeer(chatId)
	if err != nil {
		return nil, err
	}

	var newCall tg.InputGroupCall
	switch chatPeer := peer.(type) {
	case *tg.InputPeerChannel:
		fullChat, err := ctx.App.ChannelsGetFullChannel(&tg.InputChannelObj{ChannelID: chatPeer.ChannelID, AccessHash: chatPeer.AccessHash})
		if err != nil {
			return nil, err
		}
		newCall = fullChat.FullChat.(*tg.ChannelFull).Call
	case *tg.InputPeerChat:
		fullChat, err := ctx.App.MessagesGetFullChat(chatPeer.ChatID)
		if err != nil {
			return nil, err
		}
		newCall = fullChat.FullChat.(*tg.ChatFullObj).Call
	default:
		return nil, fmt.Errorf("chatId %d is not a group call", chatId)
	}

	ctx.groupCallsMutex.Lock()
	ctx.inputGroupCalls[chatId] = newCall
	if callObj, ok := newCall.(*tg.InputGroupCallObj); ok {
		ctx.groupCallIds[callObj.ID] = chatId
	}
	ctx.groupCallsMutex.Unlock()

	if newCall == nil {
		return nil, fmt.Errorf("group call for chatId %d is closed", chatId)
	}
	return newCall, nil
}
