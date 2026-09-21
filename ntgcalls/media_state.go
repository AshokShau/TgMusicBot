package ntgcalls

// MediaState represents the current mute and video pause/stop state of a call.
type MediaState struct {
	Muted              bool
	VideoPaused        bool
	VideoStopped       bool
	PresentationPaused bool
}
