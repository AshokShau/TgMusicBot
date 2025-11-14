package ubot

import "ashokshau/tgmusic/src/vc/ntgcalls"

func (ctx *Context) Time(chatId int64, streamMode ntgcalls.StreamMode) (uint64, error) {
	return ctx.binding.Time(chatId, streamMode)
}
