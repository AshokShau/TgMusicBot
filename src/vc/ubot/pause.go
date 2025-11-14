package ubot

func (ctx *Context) Pause(chatId int64) (bool, error) {
	return ctx.binding.Pause(chatId)
}
