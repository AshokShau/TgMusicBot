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

type VideoDescription struct {
	MediaSource   MediaSource
	Input         string
	Width, Height int16
	Fps           uint8
}

func (ctx *VideoDescription) ParseToC() C.ntg_video_description_struct {
	var x C.ntg_video_description_struct
	x.mediaSource = ctx.MediaSource.ParseToC()
	x.input = C.CString(ctx.Input)
	x.width = C.int16_t(ctx.Width)
	x.height = C.int16_t(ctx.Height)
	x.fps = C.uint8_t(ctx.Fps)
	return x
}
