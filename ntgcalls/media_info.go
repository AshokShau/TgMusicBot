package ntgcalls

type MediaInfo struct {
	Container  string
	DurationMs int64
	Audio      *AudioInfo
	Video      *VideoInfo
}

type AudioInfo struct {
	Codec       string
	SampleRate  uint32
	Channels    uint8
	Supported   bool
	Passthrough bool
}

type VideoInfo struct {
	Codec       string
	Width       uint16
	Height      uint16
	FrameRate   float64
	BFrames     bool
	Supported   bool
	Passthrough bool
}
