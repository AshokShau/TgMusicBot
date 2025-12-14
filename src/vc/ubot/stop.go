package ubot

import "fmt"

func (ctx *Context) Stop(chatId int64) error {
	ctx.presentationsMutex.Lock()
	ctx.presentations = stdRemove(ctx.presentations, chatId)
	ctx.presentationsMutex.Unlock()

	ctx.participantsMutex.Lock()
	delete(ctx.callSources, chatId)
	ctx.participantsMutex.Unlock()

	err := ctx.binding.Stop(chatId)
	if err != nil {
		return err
	}

	ctx.groupCallsMutex.RLock()
	call := ctx.inputGroupCalls[chatId]
	ctx.groupCallsMutex.RUnlock()
	if call == nil {
		return fmt.Errorf("no active group call found for chat %v", chatId)
	}

	_, err = ctx.App.PhoneLeaveGroupCall(call, 0)
	if err != nil {
		return err
	}
	return nil
}
