package ntgcalls

// NetworkInfo describes the network connection state and kind.
type NetworkInfo struct {
	Kind  ConnectionKind
	State ConnectionState
}
