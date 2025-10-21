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

type AudioDescription struct {
	MediaSource  MediaSource
	Input        string
	SampleRate   uint32
	ChannelCount uint8
}

func (ctx *AudioDescription) ParseToC() *C.ntg_audio_description_struct {
	cStruct := (*C.ntg_audio_description_struct)(C.malloc(C.sizeof_ntg_audio_description_struct))
	cStruct.mediaSource = ctx.MediaSource.ParseToC()
	cStruct.input = C.CString(ctx.Input)
	cStruct.sampleRate = C.uint32_t(ctx.SampleRate)
	cStruct.channelCount = C.uint8_t(ctx.ChannelCount)
	return cStruct
}

func freeAudioDescription(cStruct *C.ntg_audio_description_struct) {
	C.free(unsafe.Pointer(cStruct.input))
	C.free(unsafe.Pointer(cStruct))
}
