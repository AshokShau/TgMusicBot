package ubot

func (ctx *Context) Unmute(chatId any) (bool, error) {
	parsedChatId, err := ctx.parseChatId(chatId)
	if err != nil {
		return false, err
	}
	return ctx.binding.UnMute(parsedChatId)
}
