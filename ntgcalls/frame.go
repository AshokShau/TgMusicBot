package ntgcalls

// Frame represents a single audio or video frame.
type Frame struct {
	Ssrc      uint32
	Data      []byte
	FrameData FrameData
}
