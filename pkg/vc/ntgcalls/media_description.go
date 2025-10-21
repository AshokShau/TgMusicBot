/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ntgcalls

//#include "ntgcalls.h"
import "C"
import "unsafe"

type MediaDescription struct {
	Microphone *AudioDescription
	Speaker    *AudioDescription
	Camera     *VideoDescription
	Screen     *VideoDescription
}

func (ctx *MediaDescription) ParseToC() C.ntg_media_description_struct {
	var x C.ntg_media_description_struct
	if ctx.Microphone != nil {
		x.microphone = ctx.Microphone.ParseToC()
	}
	if ctx.Speaker != nil {
		x.speaker = ctx.Speaker.ParseToC()
	}
	if ctx.Camera != nil {
		x.camera = ctx.Camera.ParseToC()
	}
	if ctx.Screen != nil {
		x.screen = ctx.Screen.ParseToC()
	}
	return x
}

func freeMediaDescription(cStruct C.ntg_media_description_struct) {
	if cStruct.microphone != nil {
		freeAudioDescription((*C.ntg_audio_description_struct)(unsafe.Pointer(cStruct.microphone)))
	}
	if cStruct.speaker != nil {
		freeAudioDescription((*C.ntg_audio_description_struct)(unsafe.Pointer(cStruct.speaker)))
	}
	if cStruct.camera != nil {
		freeVideoDescription((*C.ntg_video_description_struct)(unsafe.Pointer(cStruct.camera)))
	}
	if cStruct.screen != nil {
		freeVideoDescription((*C.ntg_video_description_struct)(unsafe.Pointer(cStruct.screen)))
	}
}
