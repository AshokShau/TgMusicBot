package ntgcalls

// RemoteSource describes a remote participant's media source.
type RemoteSource struct {
	Ssrc   uint32
	State  StreamStatus
	Device StreamDevice
}
