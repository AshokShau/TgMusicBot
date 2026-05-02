/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ntgcalls

//#include "ntgcalls.h"
import "C"

type MediaDescription struct {
	Microphone *AudioDescription
	Speaker    *AudioDescription
	Camera     *VideoDescription
	Screen     *VideoDescription
}

func (ctx *MediaDescription) ParseToC() C.ntg_media_description_struct {
	var x C.ntg_media_description_struct
	if ctx.Microphone != nil {
		x.microphone = new(ctx.Microphone.ParseToC())
	}
	if ctx.Speaker != nil {
		x.speaker = new(ctx.Speaker.ParseToC())
	}
	if ctx.Camera != nil {
		x.camera = new(ctx.Camera.ParseToC())
	}
	if ctx.Screen != nil {
		x.screen = new(ctx.Screen.ParseToC())
	}
	return x
}
