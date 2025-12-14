package ntgcalls

//#include "ntgcalls.h"
import "C"

type MediaDescription struct {
	Microphone *AudioDescription
	Speaker    *AudioDescription
	Camera     *VideoDescription
	Screen     *VideoDescription
}

func (ctx *MediaDescription) ParseToC() (C.ntg_media_description_struct, func()) {
	var x C.ntg_media_description_struct
	cleanups := make([]func(), 0)

	if ctx.Microphone != nil {
		microphone, cleanup := ctx.Microphone.ParseToC()

		ptr := new(C.ntg_audio_description_struct)
		*ptr = microphone
		x.microphone = ptr
		cleanups = append(cleanups, cleanup)
	}
	if ctx.Speaker != nil {
		speaker, cleanup := ctx.Speaker.ParseToC()
		ptr := new(C.ntg_audio_description_struct)
		*ptr = speaker
		x.speaker = ptr
		cleanups = append(cleanups, cleanup)
	}
	if ctx.Camera != nil {
		camera, cleanup := ctx.Camera.ParseToC()
		ptr := new(C.ntg_video_description_struct)
		*ptr = camera
		x.camera = ptr
		cleanups = append(cleanups, cleanup)
	}
	if ctx.Screen != nil {
		screen, cleanup := ctx.Screen.ParseToC()
		ptr := new(C.ntg_video_description_struct)
		*ptr = screen
		x.screen = ptr
		cleanups = append(cleanups, cleanup)
	}

	finalCleanup := func() {
		for _, c := range cleanups {
			c()
		}
	}

	return x, finalCleanup
}
