package ubot

func (ctx *Context) Mute(chatId int64) (bool, error) {
	return ctx.binding.Mute(chatId)
}

func (ctx *Context) Unmute(chatId int64) (bool, error) {
	return ctx.binding.UnMute(chatId)
}
