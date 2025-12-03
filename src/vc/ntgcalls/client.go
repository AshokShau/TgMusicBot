package ntgcalls

import (
	"runtime/cgo"
	"unsafe"
)

type Client struct {
	ptr                         uintptr
	handle                      cgo.Handle
	handlePtr                   unsafe.Pointer
	connectionChangeCallbacks   []ConnectionChangeCallback
	streamEndCallbacks          []StreamEndCallback
	upgradeCallbacks            []UpgradeCallback
	signalCallbacks             []SignalCallback
	frameCallbacks              []FrameCallback
	remoteSourceCallbacks       []RemoteSourceCallback
	broadcastTimestampCallbacks []BroadcastTimestampCallback
	broadcastPartCallbacks      []BroadcastPartCallback
}
