package ubot

func (ctx *Context) Stop(chatId int64) error {
	ctx.presentations = stdRemove(ctx.presentations, chatId)
	delete(ctx.callSources, chatId)
	err := ctx.binding.Stop(chatId)
	if err != nil {
		return err
	}
	_, err = ctx.App.PhoneLeaveGroupCall(ctx.inputGroupCalls[chatId], 0)
	if err != nil {
		return err
	}
	return nil
}
