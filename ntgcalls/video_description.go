package ntgcalls

//#include "ntgcalls.h"
//#include <stdlib.h>
import "C"

// VideoDescription represents the video configuration for a media stream.
type VideoDescription struct {
	MediaSource MediaSource
	Input       string
	Width, Height int16
	Fps           uint8
	KeepOpen      bool
}

// ParseToC converts VideoDescription to the C ntg_video_description structure.
func (ctx *VideoDescription) ParseToC() C.ntg_video_description {
	var x C.ntg_video_description
	x.media_source = ctx.MediaSource.ParseToC()
	x.input = C.CString(ctx.Input)
	x.width = C.int16_t(ctx.Width)
	x.height = C.int16_t(ctx.Height)
	x.fps = C.uint8_t(ctx.Fps)
	x.keep_open = C.bool(ctx.KeepOpen)
	return x
}
