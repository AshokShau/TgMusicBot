package ubot

func (ctx *Context) Unmute(chatId int64) (bool, error) {
	return ctx.binding.UnMute(chatId)
}
