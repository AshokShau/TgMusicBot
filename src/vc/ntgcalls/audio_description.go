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
