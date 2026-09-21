package ntgcalls

//#include "ntgcalls.h"
import "C"

// MediaDescription holds the audio and video descriptions for different stream targets.
type MediaDescription struct {
	Microphone *AudioDescription
	Speaker    *AudioDescription
	Camera     *VideoDescription
	Screen     *VideoDescription
}

// ParseToC converts MediaDescription to the C ntg_media_description structure.
func (ctx *MediaDescription) ParseToC() C.ntg_media_description {
	var x C.ntg_media_description
	if ctx.Microphone != nil {
		microphone := ctx.Microphone.ParseToC()
		x.microphone = &microphone
	}
	if ctx.Speaker != nil {
		speaker := ctx.Speaker.ParseToC()
		x.speaker = &speaker
	}
	if ctx.Camera != nil {
		camera := ctx.Camera.ParseToC()
		x.camera = &camera
	}
	if ctx.Screen != nil {
		screen := ctx.Screen.ParseToC()
		x.screen = &screen
	}
	return x
}
