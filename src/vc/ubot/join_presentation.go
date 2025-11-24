package ubot

import (
	"ashokshau/tgmusic/src/vc/ntgcalls"
	"fmt"
	"slices"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) joinPresentation(chatId int64, join bool) error {
	defer func() {
		ctx.waitConnMutex.Lock()
		if ctx.waitConnect[chatId] != nil {
			delete(ctx.waitConnect, chatId)
		}
		ctx.waitConnMutex.Unlock()
	}()
	connectionMode, err := ctx.binding.GetConnectionMode(chatId)
	if err != nil {
		return err
	}
	if connectionMode == ntgcalls.StreamConnection {
		ctx.pendingConnMutex.Lock()
		if ctx.pendingConnections[chatId] != nil {
			ctx.pendingConnections[chatId].Presentation = join
		}
		ctx.pendingConnMutex.Unlock()
	} else if connectionMode == ntgcalls.RtcConnection {
		if join {
			ctx.presentationsMutex.RLock()
			exists := slices.Contains(ctx.presentations, chatId)
			ctx.presentationsMutex.RUnlock()
			if !exists {
				ctx.waitConnMutex.Lock()
				ctx.waitConnect[chatId] = make(chan error, 1)
				ctx.waitConnMutex.Unlock()

				jsonParams, err := ctx.binding.InitPresentation(chatId)
				if err != nil {
					return err
				}
				resultParams := "{\"transport\": null}"
				ctx.groupCallsMutex.RLock()
				call := ctx.inputGroupCalls[chatId]
				ctx.groupCallsMutex.RUnlock()
				callResRaw, err := ctx.App.PhoneJoinGroupCallPresentation(
					call,
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

				ctx.waitConnMutex.RLock()
				ch := ctx.waitConnect[chatId]
				ctx.waitConnMutex.RUnlock()

				select {
				case err := <-ch:
					if err != nil {
						return err
					}
				case <-time.After(15 * time.Second):
					return fmt.Errorf("timed out waiting for connection")
				}

				ctx.presentationsMutex.Lock()
				ctx.presentations = append(ctx.presentations, chatId)
				ctx.presentationsMutex.Unlock()
			}
		} else {
			ctx.presentationsMutex.RLock()
			exists := slices.Contains(ctx.presentations, chatId)
			ctx.presentationsMutex.RUnlock()
			if exists {
				ctx.presentationsMutex.Lock()
				ctx.presentations = stdRemove(ctx.presentations, chatId)
				ctx.presentationsMutex.Unlock()

				err = ctx.binding.StopPresentation(chatId)
				if err != nil {
					return err
				}
				ctx.groupCallsMutex.RLock()
				call := ctx.inputGroupCalls[chatId]
				ctx.groupCallsMutex.RUnlock()
				_, err = ctx.App.PhoneLeaveGroupCallPresentation(
					call,
				)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
