/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ntgcalls

//#include "ntgcalls.h"
//#include <stdlib.h>
import "C"
import (
	"errors"
	"unsafe"
)

type VideoDescription struct {
	MediaSource   MediaSource
	Input         string
	Width, Height int16
	Fps           uint8
	KeepOpen      bool
}

func (ctx *VideoDescription) ParseToC() C.ntg_video_description_struct {
	var x C.ntg_video_description_struct
	x.mediaSource = ctx.MediaSource.ParseToC()
	x.input = C.CString(ctx.Input)
	x.width = C.int16_t(ctx.Width)
	x.height = C.int16_t(ctx.Height)
	x.fps = C.uint8_t(ctx.Fps)
	x.keepOpen = C.bool(ctx.KeepOpen)
	return x
}

func (ctx *VideoDescription) allocC() (*C.ntg_video_description_struct, func(), error) {
	cDesc := (*C.ntg_video_description_struct)(C.malloc(C.size_t(unsafe.Sizeof(C.ntg_video_description_struct{}))))
	if cDesc == nil {
		return nil, nil, errors.New("ntgcalls: failed to allocate video description")
	}
	input := C.CString(ctx.Input)
	*cDesc = C.ntg_video_description_struct{
		mediaSource: ctx.MediaSource.ParseToC(),
		input:       input,
		width:       C.int16_t(ctx.Width),
		height:      C.int16_t(ctx.Height),
		fps:         C.uint8_t(ctx.Fps),
		keepOpen:    C.bool(ctx.KeepOpen),
	}
	cleanup := func() {
		C.free(unsafe.Pointer(input))
		C.free(unsafe.Pointer(cDesc))
	}
	return cDesc, cleanup, nil
}
