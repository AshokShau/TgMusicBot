/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ntgcalls

//#include "ntgcalls.h"
//#include <stdlib.h>
import "C"
import "unsafe"

type VideoDescription struct {
	MediaSource   MediaSource
	Input         string
	Width, Height int16
	Fps           uint8
}

func (ctx *VideoDescription) ParseToC() *C.ntg_video_description_struct {
	cStruct := (*C.ntg_video_description_struct)(C.malloc(C.sizeof_ntg_video_description_struct))
	cStruct.mediaSource = ctx.MediaSource.ParseToC()
	cStruct.input = C.CString(ctx.Input)
	cStruct.width = C.int16_t(ctx.Width)
	cStruct.height = C.int16_t(ctx.Height)
	cStruct.fps = C.uint8_t(ctx.Fps)
	return cStruct
}

func freeVideoDescription(cStruct *C.ntg_video_description_struct) {
	C.free(unsafe.Pointer(cStruct.input))
	C.free(unsafe.Pointer(cStruct))
}
