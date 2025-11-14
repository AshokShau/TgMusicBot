package ubot

import "ashokshau/tgmusic/src/vc/ntgcalls"

func (ctx *Context) Record(chatId int64, mediaDescription ntgcalls.MediaDescription) error {
	if ctx.binding.Calls()[chatId] == nil {
		err := ctx.Play(chatId, ntgcalls.MediaDescription{})
		if err != nil {
			return err
		}
	}
	return ctx.binding.SetStreamSources(chatId, ntgcalls.PlaybackStream, mediaDescription)
}
