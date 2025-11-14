package ubot

import (
	"fmt"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) getInputGroupCall(chatId int64) (tg.InputGroupCall, error) {
    ctx.groupCallsMutex.RLock()
    if call, ok := ctx.inputGroupCalls[chatId]; ok {
        if call == nil {
            ctx.groupCallsMutex.RUnlock()
            return nil, fmt.Errorf("group call for chatId %d is closed", chatId)
        }
        ctx.groupCallsMutex.RUnlock()
        return call, nil
    }
    // not found in cache, release lock before network calls
    ctx.groupCallsMutex.RUnlock()
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
        ctx.groupCallsMutex.Lock()
        ctx.inputGroupCalls[chatId] = fullChat.FullChat.(*tg.ChannelFull).Call
        ctx.groupCallsMutex.Unlock()
    case *tg.InputPeerChat:
        fullChat, err := ctx.App.MessagesGetFullChat(chatPeer.ChatID)
        if err != nil {
            return nil, err
        }
        ctx.groupCallsMutex.Lock()
        ctx.inputGroupCalls[chatId] = fullChat.FullChat.(*tg.ChatFullObj).Call
        ctx.groupCallsMutex.Unlock()
    default:
        return nil, fmt.Errorf("chatId %d is not a group call", chatId)
    }
    ctx.groupCallsMutex.RLock()
    if call, ok := ctx.inputGroupCalls[chatId]; ok && call == nil {
        ctx.groupCallsMutex.RUnlock()
        return nil, fmt.Errorf("group call for chatId %d is closed", chatId)
    } else if ok {
        ctx.groupCallsMutex.RUnlock()
        return call, nil
    } else {
        ctx.groupCallsMutex.RUnlock()
        return nil, fmt.Errorf("group call for chatId %d not found", chatId)
    }
}
