package ntgcalls

// Protocol represents WebRTC/network protocol capabilities and versions.
type Protocol struct {
	MinLayer     int32
	MaxLayer     int32
	UdpReflector bool
	Versions     []string
}
