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

func (ctx *MediaDescription) allocateC() (C.ntg_media_description_struct, []StreamDevice, func(), error) {
	var desc C.ntg_media_description_struct
	var devices []StreamDevice
	cleanups := make([]func(), 0, 4)

	cleanupAll := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	if ctx.Microphone != nil {
		audioDesc, cleanup, err := ctx.Microphone.allocC()
		if err != nil {
			cleanupAll()
			return C.ntg_media_description_struct{}, nil, nil, err
		}
		desc.microphone = audioDesc
		devices = append(devices, MicrophoneStream)
		cleanups = append(cleanups, cleanup)
	}

	if ctx.Speaker != nil {
		audioDesc, cleanup, err := ctx.Speaker.allocC()
		if err != nil {
			cleanupAll()
			return C.ntg_media_description_struct{}, nil, nil, err
		}
		desc.speaker = audioDesc
		devices = append(devices, SpeakerStream)
		cleanups = append(cleanups, cleanup)
	}

	if ctx.Camera != nil {
		videoDesc, cleanup, err := ctx.Camera.allocC()
		if err != nil {
			cleanupAll()
			return C.ntg_media_description_struct{}, nil, nil, err
		}
		desc.camera = videoDesc
		devices = append(devices, CameraStream)
		cleanups = append(cleanups, cleanup)
	}

	if ctx.Screen != nil {
		videoDesc, cleanup, err := ctx.Screen.allocC()
		if err != nil {
			cleanupAll()
			return C.ntg_media_description_struct{}, nil, nil, err
		}
		desc.screen = videoDesc
		devices = append(devices, ScreenStream)
		cleanups = append(cleanups, cleanup)
	}

	return desc, devices, cleanupAll, nil
}
