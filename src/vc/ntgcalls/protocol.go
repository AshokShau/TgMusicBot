package ntgcalls

type Protocol struct {
	MinLayer     int32
	MaxLayer     int32
	UdpReflector bool
	Versions     []string
}
