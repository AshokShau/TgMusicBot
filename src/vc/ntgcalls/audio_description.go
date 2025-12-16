package ntgcalls

//#include "ntgcalls.h"
//#include <stdlib.h>
import "C"
import "unsafe"

type AudioDescription struct {
	MediaSource  MediaSource
	Input        string
	SampleRate   uint32
	ChannelCount uint8
	KeepOpen     bool
}

func (ctx *AudioDescription) ParseToC() (C.ntg_audio_description_struct, func()) {
	var x C.ntg_audio_description_struct
	x.mediaSource = ctx.MediaSource.ParseToC()
	x.input = C.CString(ctx.Input)
	x.sampleRate = C.uint32_t(ctx.SampleRate)
	x.channelCount = C.uint8_t(ctx.ChannelCount)
	x.keepOpen = C.bool(ctx.KeepOpen)
	return x, func() {
		C.free(unsafe.Pointer(x.input))
	}
}
