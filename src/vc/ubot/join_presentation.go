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
	"fmt"
	"slices"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) joinPresentation(chatId int64, join bool) error {
	chatMutex := ctx.getChatMutex(chatId)
	chatMutex.Lock()
	defer chatMutex.Unlock()

	defer func() {
		ctx.stateMutex.Lock()
		if ctx.waitConnect[chatId] != nil {
			delete(ctx.waitConnect, chatId)
		}
		ctx.stateMutex.Unlock()
	}()
	connectionMode, err := ctx.binding.GetConnectionMode(chatId)
	if err != nil {
		return err
	}
	if connectionMode == ntgcalls.StreamConnection {
		ctx.stateMutex.Lock()
		if ctx.pendingConnections[chatId] != nil {
			ctx.pendingConnections[chatId].Presentation = join
		}
		ctx.stateMutex.Unlock()
	} else if connectionMode == ntgcalls.RtcConnection {
		ctx.stateMutex.Lock()
		isJoined := slices.Contains(ctx.presentations, chatId)
		ctx.stateMutex.Unlock()

		if join {
			if !isJoined {
				ctx.stateMutex.Lock()
				if ctx.waitConnect[chatId] != nil {
					ctx.stateMutex.Unlock()
					return fmt.Errorf("connection already in progress for chat %d", chatId)
				}
				ctx.waitConnect[chatId] = make(chan error, 1) // Buffered to prevent deadlock
				ctx.stateMutex.Unlock()
				jsonParams, err := ctx.binding.InitPresentation(chatId)
				if err != nil {
					return err
				}
				resultParams := "{\"transport\": null}"
				ctx.inputGroupCallsMutex.RLock()
				inputGroupCall := ctx.inputGroupCalls[chatId]
				ctx.inputGroupCallsMutex.RUnlock()
				callResRaw, err := ctx.App.PhoneJoinGroupCallPresentation(
					inputGroupCall,
					&tg.DataJson{
						Data: jsonParams,
					},
				)
				if err != nil {
					return err
				}
				callRes := callResRaw.(*tg.UpdatesObj)
				for _, update := range callRes.Updates {
					switch update.(type) {
					case *tg.UpdateGroupCallConnection:
						resultParams = update.(*tg.UpdateGroupCallConnection).Params.Data
					}
				}
				err = ctx.binding.Connect(
					chatId,
					resultParams,
					true,
				)
				if err != nil {
					return err
				}
				ctx.stateMutex.Lock()
				waitChan := ctx.waitConnect[chatId]
				ctx.stateMutex.Unlock()
				chatMutex.Unlock()
				<-waitChan
				chatMutex.Lock()
				ctx.stateMutex.Lock()
				ctx.presentations = append(ctx.presentations, chatId)
				ctx.stateMutex.Unlock()
			}
		} else if isJoined {
			ctx.stateMutex.Lock()
			ctx.presentations = stdRemove(ctx.presentations, chatId)
			ctx.stateMutex.Unlock()
			err = ctx.binding.StopPresentation(chatId)
			if err != nil {
				return err
			}
			ctx.inputGroupCallsMutex.RLock()
			inputGroupCall := ctx.inputGroupCalls[chatId]
			ctx.inputGroupCallsMutex.RUnlock()
			_, err = ctx.App.PhoneLeaveGroupCallPresentation(
				inputGroupCall,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
