package ubot

func (ctx *Context) Resume(chatId int64) (bool, error) {
	return ctx.binding.Resume(chatId)
}
