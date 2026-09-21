package ntgcalls

// CallInfo contains the playback and capture status of a call.
type CallInfo struct {
	Playback StreamStatus
	Capture  StreamStatus
}
