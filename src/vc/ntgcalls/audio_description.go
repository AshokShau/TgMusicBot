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

type AudioDescription struct {
	MediaSource  MediaSource
	Input        string
	SampleRate   uint32
	ChannelCount uint8
	KeepOpen     bool
}

func (ctx *AudioDescription) ParseToC() C.ntg_audio_description_struct {
	var x C.ntg_audio_description_struct
	x.mediaSource = ctx.MediaSource.ParseToC()
	x.input = C.CString(ctx.Input)
	x.sampleRate = C.uint32_t(ctx.SampleRate)
	x.channelCount = C.uint8_t(ctx.ChannelCount)
	x.keepOpen = C.bool(ctx.KeepOpen)
	return x
}

func (ctx *AudioDescription) allocC() (*C.ntg_audio_description_struct, func(), error) {
	cDesc := (*C.ntg_audio_description_struct)(C.malloc(C.size_t(unsafe.Sizeof(C.ntg_audio_description_struct{}))))
	if cDesc == nil {
		return nil, nil, errors.New("ntgcalls: failed to allocate audio description")
	}
	input := C.CString(ctx.Input)
	*cDesc = C.ntg_audio_description_struct{
		mediaSource:  ctx.MediaSource.ParseToC(),
		input:        input,
		sampleRate:   C.uint32_t(ctx.SampleRate),
		channelCount: C.uint8_t(ctx.ChannelCount),
		keepOpen:     C.bool(ctx.KeepOpen),
	}
	cleanup := func() {
		C.free(unsafe.Pointer(input))
		C.free(unsafe.Pointer(cDesc))
	}
	return cDesc, cleanup, nil
}
