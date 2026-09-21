package ntgcalls

//#include "ntgcalls.h"
import "C"

// Client is the Go wrapper around the C ntgcalls instance and its registered callbacks.
type Client struct {
	handle                      *C.ntg_instance
	connectionChangeCallbacks   []ConnectionChangeCallback
	streamEndCallbacks          []StreamEndCallback
	upgradeCallbacks            []UpgradeCallback
	frameCallbacks              []FrameCallback
	remoteSourceCallbacks       []RemoteSourceCallback
	broadcastTimestampCallbacks []BroadcastTimestampCallback
	broadcastPartCallbacks      []BroadcastPartCallback
}
