package ntgcalls

//#include "ntgcalls.h"
//#include <stdlib.h>
//extern void handleStreamEnd(uintptr_t ptr, int64_t chatID, ntg_stream_type_enum streamType, ntg_stream_device_enum streamDevice, void*);
//extern void handleUpgrade(uintptr_t ptr, int64_t chatID, ntg_media_state_struct state, void*);
//extern void handleConnectionChange(uintptr_t ptr, int64_t chatID, ntg_network_info_struct networkInfo, void*);
//extern void handleSignal(uintptr_t ptr, int64_t chatID, uint8_t*, int, void*);
//extern void handleFrames(uintptr_t ptr, int64_t chatID, ntg_stream_mode_enum streamMode, ntg_stream_device_enum streamDevice, ntg_frame_struct* frames, int size, void*);
//extern void handleRemoteSourceChange(uintptr_t ptr, int64_t chatID, ntg_remote_source_struct remoteSource, void*);
//extern void handleRequestBroadcastTimestamp(uintptr_t ptr, int64_t chatID, void*);
//extern void handleRequestBroadcastPart(uintptr_t ptr, int64_t chatID, ntg_segment_part_request_struct segmentPartRequest, void*);
//extern void handleLogs(ntg_log_message_struct logMessage);
import "C"
import (
	"fmt"
	"runtime/cgo"
	"runtime/debug"
	"unsafe"
)

func init() {
	C.ntg_register_logger((C.ntg_log_message_callback)(unsafe.Pointer(C.handleLogs)))
}

func NTgCalls() *Client {
	instance := &Client{
		ptr: uintptr(C.ntg_init()),
	}
	h := cgo.NewHandle(instance)
	instance.handle = h

	ptr := C.malloc(C.size_t(unsafe.Sizeof(h)))
	*(*cgo.Handle)(ptr) = h
	instance.handlePtr = ptr

	selfPointer := instance.handlePtr

	C.ntg_on_stream_end(C.uintptr_t(instance.ptr), (C.ntg_stream_callback)(unsafe.Pointer(C.handleStreamEnd)), selfPointer)
	C.ntg_on_upgrade(C.uintptr_t(instance.ptr), (C.ntg_upgrade_callback)(unsafe.Pointer(C.handleUpgrade)), selfPointer)
	C.ntg_on_signaling_data(C.uintptr_t(instance.ptr), (C.ntg_signaling_callback)(unsafe.Pointer(C.handleSignal)), selfPointer)
	C.ntg_on_connection_change(C.uintptr_t(instance.ptr), (C.ntg_connection_callback)(unsafe.Pointer(C.handleConnectionChange)), selfPointer)
	C.ntg_on_frames(C.uintptr_t(instance.ptr), (C.ntg_frame_callback)(unsafe.Pointer(C.handleFrames)), selfPointer)
	C.ntg_on_remote_source_change(C.uintptr_t(instance.ptr), (C.ntg_remote_source_callback)(unsafe.Pointer(C.handleRemoteSourceChange)), selfPointer)
	C.ntg_on_request_broadcast_timestamp(C.uintptr_t(instance.ptr), (C.ntg_broadcast_timestamp_callback)(unsafe.Pointer(C.handleRequestBroadcastTimestamp)), selfPointer)
	C.ntg_on_request_broadcast_part(C.uintptr_t(instance.ptr), (C.ntg_broadcast_part_callback)(unsafe.Pointer(C.handleRequestBroadcastPart)), selfPointer)
	return instance
}

var (
	loggerNTGCalls = NewLogger("ntgcalls", LevelInfo)
	loggerWebRTC   = NewLogger("webrtc", LevelFatal)
)

func recoverPanic() {
	if r := recover(); r != nil {
		loggerNTGCalls.Error(fmt.Sprintf("Panic in ntgcalls callback: %v\nStack: %s", r, debug.Stack()))
	}
}

//export handleLogs
func handleLogs(logMessage C.ntg_log_message_struct) {
	defer recoverPanic()
	message := fmt.Sprintf(
		"(%v:%v) %v",
		C.GoString(logMessage.file),
		uint32(logMessage.line),
		C.GoString(logMessage.message),
	)

	var lg *Logger
	if logMessage.source == C.NTG_LOG_WEBRTC {
		lg = loggerWebRTC
	} else {
		lg = loggerNTGCalls
	}

	switch logMessage.level {
	case C.NTG_LOG_DEBUG:
		lg.Debug(message)
	case C.NTG_LOG_INFO:
		lg.Info(message)
	case C.NTG_LOG_WARNING:
		lg.Warn(message)
	case C.NTG_LOG_ERROR:
		lg.Error(message)
	}
}

//export handleStreamEnd
func handleStreamEnd(_ C.uintptr_t, chatID C.int64_t, streamType C.ntg_stream_type_enum, streamDevice C.ntg_stream_device_enum, ptr unsafe.Pointer) {
	defer recoverPanic()
	h := *(*cgo.Handle)(ptr)
	self := h.Value().(*Client)
	goChatID := int64(chatID)
	var goStreamType StreamType
	if streamType == C.NTG_STREAM_AUDIO {
		goStreamType = AudioStream
	} else {
		goStreamType = VideoStream
	}
	self.mu.RLock()
	for _, x0 := range self.streamEndCallbacks {
		go func(cb StreamEndCallback) {
			defer recoverPanic()
			cb(goChatID, goStreamType, parseStreamDevice(streamDevice))
		}(x0)
	}
	self.mu.RUnlock()
}

//export handleUpgrade
func handleUpgrade(_ C.uintptr_t, chatID C.int64_t, state C.ntg_media_state_struct, ptr unsafe.Pointer) {
	defer recoverPanic()
	h := *(*cgo.Handle)(ptr)
	self := h.Value().(*Client)
	goChatID := int64(chatID)
	goState := MediaState{
		Muted:              bool(state.muted),
		VideoPaused:        bool(state.videoPaused),
		VideoStopped:       bool(state.videoStopped),
		PresentationPaused: bool(state.presentationPaused),
	}
	self.mu.RLock()
	for _, x0 := range self.upgradeCallbacks {
		go func(cb UpgradeCallback) {
			defer recoverPanic()
			cb(goChatID, goState)
		}(x0)
	}
	self.mu.RUnlock()
}

//export handleSignal
func handleSignal(_ C.uintptr_t, chatID C.int64_t, data *C.uint8_t, size C.int, ptr unsafe.Pointer) {
	defer recoverPanic()
	h := *(*cgo.Handle)(ptr)
	self := h.Value().(*Client)
	goChatID := int64(chatID)
	self.mu.RLock()
	for _, x0 := range self.signalCallbacks {
		go func(cb SignalCallback) {
			defer recoverPanic()
			cb(goChatID, C.GoBytes(unsafe.Pointer(data), size))
		}(x0)
	}
	self.mu.RUnlock()
}

//export handleConnectionChange
func handleConnectionChange(_ C.uintptr_t, chatID C.int64_t, networkInfo C.ntg_network_info_struct, ptr unsafe.Pointer) {
	defer recoverPanic()
	h := *(*cgo.Handle)(ptr)
	self := h.Value().(*Client)
	goChatID := int64(chatID)
	var goCallState NetworkInfo
	switch networkInfo.kind {
	case C.NTG_KIND_NORMAL:
		goCallState.Kind = NormalConnection
	case C.NTG_KIND_PRESENTATION:
		goCallState.Kind = PresentationConnection
	}
	goCallState.State = parseConnectionState(networkInfo.state)
	self.mu.RLock()
	for _, x0 := range self.connectionChangeCallbacks {
		go func(cb ConnectionChangeCallback) {
			defer recoverPanic()
			cb(goChatID, goCallState)
		}(x0)
	}
	self.mu.RUnlock()
}

//export handleFrames
func handleFrames(_ C.uintptr_t, chatID C.int64_t, streamMode C.ntg_stream_mode_enum, streamDevice C.ntg_stream_device_enum, frames *C.ntg_frame_struct, size C.int, ptr unsafe.Pointer) {
	defer recoverPanic()
	h := *(*cgo.Handle)(ptr)
	self := h.Value().(*Client)
	goChatID := int64(chatID)
	var goStreamMode StreamMode
	switch streamMode {
	case C.NTG_STREAM_CAPTURE:
		goStreamMode = CaptureStream
	case C.NTG_STREAM_PLAYBACK:
		goStreamMode = PlaybackStream
	}
	rawFrames := make([]Frame, size)
	for i := 0; i < int(size); i++ {
		rawFrame := *(*C.ntg_frame_struct)(unsafe.Pointer(uintptr(unsafe.Pointer(frames)) + uintptr(i)*unsafe.Sizeof(C.ntg_frame_struct{})))
		rawFrames[i] = Frame{
			Ssrc: uint32(rawFrame.ssrc),
			Data: C.GoBytes(unsafe.Pointer(rawFrame.data), rawFrame.sizeData),
			FrameData: FrameData{
				AbsoluteCaptureTimestampMs: int64(frames.frameData.absoluteCaptureTimestampMs),
				Width:                      uint16(frames.frameData.width),
				Height:                     uint16(frames.frameData.height),
				Rotation:                   uint16(frames.frameData.rotation),
			},
		}
	}
	self.mu.RLock()
	for _, x0 := range self.frameCallbacks {
		go func(cb FrameCallback) {
			defer recoverPanic()
			cb(goChatID, goStreamMode, parseStreamDevice(streamDevice), rawFrames)
		}(x0)
	}
	self.mu.RUnlock()
}

//export handleRemoteSourceChange
func handleRemoteSourceChange(_ C.uintptr_t, chatID C.int64_t, remoteSource C.ntg_remote_source_struct, ptr unsafe.Pointer) {
	defer recoverPanic()
	h := *(*cgo.Handle)(ptr)
	self := h.Value().(*Client)
	goChatID := int64(chatID)
	goRemoteSource := RemoteSource{
		Ssrc:   uint32(remoteSource.ssrc),
		State:  parseStreamStatus(remoteSource.state),
		Device: parseStreamDevice(remoteSource.device),
	}
	self.mu.RLock()
	for _, x0 := range self.remoteSourceCallbacks {
		go func(cb RemoteSourceCallback) {
			defer recoverPanic()
			cb(goChatID, goRemoteSource)
		}(x0)
	}
	self.mu.RUnlock()
}

//export handleRequestBroadcastTimestamp
func handleRequestBroadcastTimestamp(_ C.uintptr_t, chatID C.int64_t, ptr unsafe.Pointer) {
	defer recoverPanic()
	h := *(*cgo.Handle)(ptr)
	self := h.Value().(*Client)
	goChatID := int64(chatID)
	self.mu.RLock()
	for _, x0 := range self.broadcastTimestampCallbacks {
		go func(cb BroadcastTimestampCallback) {
			defer recoverPanic()
			cb(goChatID)
		}(x0)
	}
	self.mu.RUnlock()
}

//export handleRequestBroadcastPart
func handleRequestBroadcastPart(_ C.uintptr_t, chatID C.int64_t, segmentPartRequest C.ntg_segment_part_request_struct, ptr unsafe.Pointer) {
	defer recoverPanic()
	h := *(*cgo.Handle)(ptr)
	self := h.Value().(*Client)
	goChatID := int64(chatID)
	var goSegmentQuality MediaSegmentQuality
	switch segmentPartRequest.quality {
	case C.NTG_MEDIA_SEGMENT_QUALITY_NONE:
		goSegmentQuality = SegmentQualityNone
	case C.NTG_MEDIA_SEGMENT_QUALITY_THUMBNAIL:
		goSegmentQuality = SegmentQualityThumbnail
	case C.NTG_MEDIA_SEGMENT_QUALITY_MEDIUM:
		goSegmentQuality = SegmentQualityMedium
	case C.NTG_MEDIA_SEGMENT_QUALITY_FULL:
		goSegmentQuality = SegmentQualityFull
	}
	goSegmentPartRequest := SegmentPartRequest{
		SegmentID:     int64(segmentPartRequest.segmentId),
		PartID:        int32(segmentPartRequest.partId),
		Limit:         int32(segmentPartRequest.limit),
		Timestamp:     int64(segmentPartRequest.timestamp),
		QualityUpdate: bool(segmentPartRequest.qualityUpdate),
		ChannelID:     int32(segmentPartRequest.channelId),
		Quality:       goSegmentQuality,
	}
	self.mu.RLock()
	for _, x0 := range self.broadcastPartCallbacks {
		go func(cb BroadcastPartCallback) {
			defer recoverPanic()
			cb(goChatID, goSegmentPartRequest)
		}(x0)
	}
	self.mu.RUnlock()
}

func (ctx *Client) OnStreamEnd(callback StreamEndCallback) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.streamEndCallbacks = append(ctx.streamEndCallbacks, callback)
}

func (ctx *Client) OnUpgrade(callback UpgradeCallback) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.upgradeCallbacks = append(ctx.upgradeCallbacks, callback)
}

func (ctx *Client) OnConnectionChange(callback ConnectionChangeCallback) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.connectionChangeCallbacks = append(ctx.connectionChangeCallbacks, callback)
}

func (ctx *Client) OnSignal(callback SignalCallback) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.signalCallbacks = append(ctx.signalCallbacks, callback)
}

func (ctx *Client) OnFrame(callback FrameCallback) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.frameCallbacks = append(ctx.frameCallbacks, callback)
}

func (ctx *Client) OnRemoteSourceChange(callback RemoteSourceCallback) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.remoteSourceCallbacks = append(ctx.remoteSourceCallbacks, callback)
}

func (ctx *Client) OnRequestBroadcastTimestamp(callback BroadcastTimestampCallback) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.broadcastTimestampCallbacks = append(ctx.broadcastTimestampCallbacks, callback)
}

func (ctx *Client) OnRequestBroadcastPart(callback BroadcastPartCallback) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.broadcastPartCallbacks = append(ctx.broadcastPartCallbacks, callback)
}

func (ctx *Client) GetState(chatId int64) (MediaState, error) {
	f := CreateFuture()
	defer f.Free()
	bufferPtr := (*C.ntg_media_state_struct)(C.malloc(C.size_t(unsafe.Sizeof(C.ntg_media_state_struct{}))))
	defer C.free(unsafe.Pointer(bufferPtr))
	C.ntg_get_state(C.uintptr_t(ctx.ptr), C.int64_t(chatId), bufferPtr, f.ParseToC())
	f.wait()
	err := parseErrorCode(f)
	if err != nil {
		return MediaState{}, err
	}
	buffer := *bufferPtr
	return MediaState{
		Muted:              bool(buffer.muted),
		VideoPaused:        bool(buffer.videoPaused),
		VideoStopped:       bool(buffer.videoStopped),
		PresentationPaused: bool(buffer.presentationPaused),
	}, nil
}

func (ctx *Client) GetConnectionMode(chatId int64) (ConnectionMode, error) {
	f := CreateFuture()
	defer f.Free()
	bufferPtr := (*C.ntg_connection_mode_enum)(C.malloc(C.size_t(unsafe.Sizeof(C.ntg_connection_mode_enum(0)))))
	defer C.free(unsafe.Pointer(bufferPtr))
	C.ntg_get_connection_mode(C.uintptr_t(ctx.ptr), C.int64_t(chatId), bufferPtr, f.ParseToC())
	f.wait()
	err := parseErrorCode(f)
	if err != nil {
		return ConnectionMode(0), err
	}
	buffer := *bufferPtr
	switch buffer {
	case C.NTG_CONNECTION_MODE_RTC:
		return RtcConnection, nil
	case C.NTG_CONNECTION_MODE_STREAM:
		return StreamConnection, nil
	case C.NTG_CONNECTION_MODE_RTMP:
		return RTMPConnection, nil
	default:
		return ConnectionMode(0), fmt.Errorf("unknown connection mode")
	}
}

func (ctx *Client) CreateCall(chatId int64) (string, error) {
	f := CreateFuture()
	defer f.Free()
	bufferPtr := (**C.char)(C.malloc(C.size_t(unsafe.Sizeof((*C.char)(nil)))))
	*bufferPtr = nil
	defer C.free(unsafe.Pointer(bufferPtr))
	C.ntg_create(C.uintptr_t(ctx.ptr), C.int64_t(chatId), bufferPtr, f.ParseToC())
	f.wait()
	if err := parseErrorCode(f); err != nil {
		return "", err
	}
	buffer := *bufferPtr
	defer C.free(unsafe.Pointer(buffer))
	return C.GoString(buffer), nil
}

func (ctx *Client) InitPresentation(chatId int64) (string, error) {
	f := CreateFuture()
	defer f.Free()
	bufferPtr := (**C.char)(C.malloc(C.size_t(unsafe.Sizeof((*C.char)(nil)))))
	*bufferPtr = nil
	defer C.free(unsafe.Pointer(bufferPtr))
	C.ntg_init_presentation(C.uintptr_t(ctx.ptr), C.int64_t(chatId), bufferPtr, f.ParseToC())
	f.wait()
	if err := parseErrorCode(f); err != nil {
		return "", err
	}
	buffer := *bufferPtr
	defer C.free(unsafe.Pointer(buffer))
	return C.GoString(buffer), nil
}

func (ctx *Client) StopPresentation(chatId int64) error {
	f := CreateFuture()
	defer f.Free()
	C.ntg_stop_presentation(C.uintptr_t(ctx.ptr), C.int64_t(chatId), f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

func (ctx *Client) AddIncomingVideo(chatId int64, endpoint string, ssrcGroups []SsrcGroup) (uint32, error) {
	bufferPtr := (*C.uint32_t)(C.malloc(C.size_t(unsafe.Sizeof(C.uint32_t(0)))))
	defer C.free(unsafe.Pointer(bufferPtr))
	f := CreateFuture()
	defer f.Free()

	endpointC := C.CString(endpoint)
	defer C.free(unsafe.Pointer(endpointC))

	groupsC, cleanup := parseSsrcGroups(ssrcGroups)
	defer cleanup()

	C.ntg_add_incoming_video(C.uintptr_t(ctx.ptr), C.int64_t(chatId), endpointC, groupsC, C.int(len(ssrcGroups)), bufferPtr, f.ParseToC())
	f.wait()
	if err := parseErrorCode(f); err != nil {
		return 0, err
	}
	return uint32(*bufferPtr), nil
}

func (ctx *Client) RemoveIncomingVideo(chatId int64, endpoint string) error {
	f := CreateFuture()
	defer f.Free()

	endpointC := C.CString(endpoint)
	defer C.free(unsafe.Pointer(endpointC))

	C.ntg_remove_incoming_video(C.uintptr_t(ctx.ptr), C.int64_t(chatId), endpointC, f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

func (ctx *Client) CreateP2PCall(chatId int64) error {
	f := CreateFuture()
	defer f.Free()
	C.ntg_create_p2p(C.uintptr_t(ctx.ptr), C.int64_t(chatId), f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

func (ctx *Client) InitExchange(chatId int64, dhConfig DhConfig, gAHash []byte) ([]byte, error) {
	bufferPtr := (**C.uint8_t)(C.malloc(C.size_t(unsafe.Sizeof((*C.uint8_t)(nil)))))
	*bufferPtr = nil
	defer C.free(unsafe.Pointer(bufferPtr))
	sizePtr := (*C.int)(C.malloc(C.size_t(unsafe.Sizeof(C.int(0)))))
	*sizePtr = 0
	defer C.free(unsafe.Pointer(sizePtr))

	gAHashC, gAHashSize, cleanup := parseBytes(gAHash)
	defer cleanup()

	dhConfigC, cleanupDh := dhConfig.ParseToC()
	defer cleanupDh()

	f := CreateFuture()
	defer f.Free()
	C.ntg_init_exchange(C.uintptr_t(ctx.ptr), C.int64_t(chatId), &dhConfigC, gAHashC, gAHashSize, bufferPtr, sizePtr, f.ParseToC())
	f.wait()
	if err := parseErrorCode(f); err != nil {
		return nil, err
	}
	buffer := *bufferPtr
	size := *sizePtr
	defer C.free(unsafe.Pointer(buffer))
	return C.GoBytes(unsafe.Pointer(buffer), size), nil
}

func (ctx *Client) ExchangeKeys(chatId int64, gAB []byte, fingerprint int64) (AuthParams, error) {
	f := CreateFuture()
	defer f.Free()
	bufferPtr := (*C.ntg_auth_params_struct)(C.malloc(C.size_t(unsafe.Sizeof(C.ntg_auth_params_struct{}))))
	defer C.free(unsafe.Pointer(bufferPtr))
	gABC, gABSize, cleanup := parseBytes(gAB)
	defer cleanup()

	C.ntg_exchange_keys(C.uintptr_t(ctx.ptr), C.int64_t(chatId), gABC, gABSize, C.int64_t(fingerprint), bufferPtr, f.ParseToC())
	f.wait()
	if err := parseErrorCode(f); err != nil {
		return AuthParams{}, err
	}
	buffer := *bufferPtr
	return AuthParams{
		GAOrB:          C.GoBytes(unsafe.Pointer(buffer.g_a_or_b), buffer.sizeGAB),
		KeyFingerprint: int64(buffer.key_fingerprint),
	}, nil
}

func (ctx *Client) SkipExchange(chatId int64, encryptionKey []byte, isOutgoing bool) error {
	f := CreateFuture()
	defer f.Free()
	encryptionKeyC, encryptionKeySize, cleanup := parseBytes(encryptionKey)
	defer cleanup()

	C.ntg_skip_exchange(C.uintptr_t(ctx.ptr), C.int64_t(chatId), encryptionKeyC, encryptionKeySize, C.bool(isOutgoing), f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

func (ctx *Client) ConnectP2P(chatId int64, rtcServers []RTCServer, versions []string, P2PAllowed bool) error {
	f := CreateFuture()
	defer f.Free()
	versionsC, sizeVersions, cleanupVersions := parseStringVectorC(versions)
	defer cleanupVersions()

	serversC, cleanupServers := parseRtcServers(rtcServers)
	defer cleanupServers()

	C.ntg_connect_p2p(C.uintptr_t(ctx.ptr), C.int64_t(chatId), serversC, C.int(len(rtcServers)), versionsC, C.int(sizeVersions), C.bool(P2PAllowed), f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

func (ctx *Client) SendSignalingData(chatId int64, data []byte) error {
	f := CreateFuture()
	defer f.Free()
	dataC, dataSize, cleanup := parseBytes(data)
	defer cleanup()

	C.ntg_send_signaling_data(C.uintptr_t(ctx.ptr), C.int64_t(chatId), dataC, dataSize, f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

//goland:noinspection GoUnusedExportedFunction
func GetProtocol() Protocol {
	var buffer C.ntg_protocol_struct
	C.ntg_get_protocol(&buffer)
	return Protocol{
		MinLayer:     int32(buffer.minLayer),
		MaxLayer:     int32(buffer.maxLayer),
		UdpP2P:       bool(buffer.udpP2P),
		UdpReflector: bool(buffer.udpReflector),
		Versions:     parseStringVector(unsafe.Pointer(buffer.libraryVersions), buffer.libraryVersionsSize),
	}
}

func (ctx *Client) Connect(chatId int64, params string, isPresentation bool) error {
	f := CreateFuture()
	defer f.Free()

	paramsC := C.CString(params)
	defer C.free(unsafe.Pointer(paramsC))

	C.ntg_connect(C.uintptr_t(ctx.ptr), C.int64_t(chatId), paramsC, C.bool(isPresentation), f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

func (ctx *Client) SetStreamSources(chatId int64, streamMode StreamMode, desc MediaDescription) error {
	f := CreateFuture()
	defer f.Free()
	descC, cleanup := desc.ParseToC()
	defer cleanup()
	C.ntg_set_stream_sources(C.uintptr_t(ctx.ptr), C.int64_t(chatId), streamMode.ParseToC(), descC, f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

func (ctx *Client) SendExternalFrame(chatId int64, streamDevice StreamDevice, data []byte, frameData FrameData) error {
	f := CreateFuture()
	defer f.Free()
	dataC, dataSize, cleanup := parseBytes(data)
	defer cleanup()

	C.ntg_send_external_frame(C.uintptr_t(ctx.ptr), C.int64_t(chatId), streamDevice.ParseToC(), dataC, dataSize, frameData.ParseToC(), f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

func (ctx *Client) SendBroadcastTimestamp(chatId int64, timestamp int64) error {
	f := CreateFuture()
	defer f.Free()
	C.ntg_send_broadcast_timestamp(C.uintptr_t(ctx.ptr), C.int64_t(chatId), C.int64_t(timestamp), f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

func (ctx *Client) SendBroadcastPart(chatId int64, segmentID int64, partID int32, status MediaSegmentStatus, qualityUpdate bool, data []byte) error {
	f := CreateFuture()
	defer f.Free()
	dataC, dataSize, cleanup := parseBytes(data)
	defer cleanup()

	C.ntg_send_broadcast_part(C.uintptr_t(ctx.ptr), C.int64_t(chatId), C.int64_t(segmentID), C.int32_t(partID), status.ParseToC(), C.bool(qualityUpdate), dataC, dataSize, f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

func (ctx *Client) Pause(chatId int64) (bool, error) {
	f := CreateFuture()
	defer f.Free()
	C.ntg_pause(C.uintptr_t(ctx.ptr), C.int64_t(chatId), f.ParseToC())
	f.wait()
	return parseBool(f)
}

func (ctx *Client) Resume(chatId int64) (bool, error) {
	f := CreateFuture()
	defer f.Free()
	C.ntg_resume(C.uintptr_t(ctx.ptr), C.int64_t(chatId), f.ParseToC())
	f.wait()
	return parseBool(f)
}

func (ctx *Client) Mute(chatId int64) (bool, error) {
	f := CreateFuture()
	defer f.Free()
	C.ntg_mute(C.uintptr_t(ctx.ptr), C.int64_t(chatId), f.ParseToC())
	f.wait()
	return parseBool(f)
}

func (ctx *Client) UnMute(chatId int64) (bool, error) {
	f := CreateFuture()
	defer f.Free()
	C.ntg_unmute(C.uintptr_t(ctx.ptr), C.int64_t(chatId), f.ParseToC())
	f.wait()
	return parseBool(f)
}

func (ctx *Client) Stop(chatId int64) error {
	f := CreateFuture()
	defer f.Free()
	C.ntg_stop(C.uintptr_t(ctx.ptr), C.int64_t(chatId), f.ParseToC())
	f.wait()
	return parseErrorCode(f)
}

func (ctx *Client) Time(chatId int64, streamMode StreamMode) (uint64, error) {
	f := CreateFuture()
	defer f.Free()
	bufferPtr := (*C.int64_t)(C.malloc(C.size_t(unsafe.Sizeof(C.int64_t(0)))))
	defer C.free(unsafe.Pointer(bufferPtr))
	C.ntg_time(C.uintptr_t(ctx.ptr), C.int64_t(chatId), streamMode.ParseToC(), bufferPtr, f.ParseToC())
	f.wait()
	if err := parseErrorCode(f); err != nil {
		return 0, err
	}
	return uint64(*bufferPtr), nil
}

//goland:noinspection GoUnusedExportedFunction
func GetMediaDevices() MediaDevices {
	var buffer C.ntg_media_devices_struct
	C.ntg_get_media_devices(&buffer)
	return MediaDevices{
		Microphone: parseDeviceInfoVector(unsafe.Pointer(buffer.microphone), buffer.sizeMicrophone),
		Speaker:    parseDeviceInfoVector(unsafe.Pointer(buffer.speaker), buffer.sizeSpeaker),
		Camera:     parseDeviceInfoVector(unsafe.Pointer(buffer.camera), buffer.sizeCamera),
		Screen:     parseDeviceInfoVector(unsafe.Pointer(buffer.screen), buffer.sizeScreen),
	}
}

func (ctx *Client) CpuUsage() (float64, error) {
	f := CreateFuture()
	defer f.Free()
	bufferPtr := (*C.double)(C.malloc(C.size_t(unsafe.Sizeof(C.double(0)))))
	defer C.free(unsafe.Pointer(bufferPtr))
	C.ntg_cpu_usage(C.uintptr_t(ctx.ptr), bufferPtr, f.ParseToC())
	f.wait()
	if err := parseErrorCode(f); err != nil {
		return 0, err
	}
	return float64(*bufferPtr), nil
}

func (ctx *Client) EnableGLibLoop(enable bool) {
	C.ntg_enable_g_lib_loop(C.bool(enable))
}

func (ctx *Client) Calls() map[int64]*CallInfo {
	mapReturn := make(map[int64]*CallInfo)
	f := CreateFuture()
	defer f.Free()
	bufferPtr := (**C.ntg_call_info_struct)(C.malloc(C.size_t(unsafe.Sizeof((*C.ntg_call_info_struct)(nil)))))
	*bufferPtr = nil
	defer C.free(unsafe.Pointer(bufferPtr))
	sizePtr := (*C.int)(C.malloc(C.size_t(unsafe.Sizeof(C.int(0)))))
	*sizePtr = 0
	defer C.free(unsafe.Pointer(sizePtr))

	C.ntg_calls(C.uintptr_t(ctx.ptr), bufferPtr, sizePtr, f.ParseToC())
	f.wait()
	if err := parseErrorCode(f); err != nil {
		return mapReturn
	}

	buffer := *bufferPtr
	size := *sizePtr
	if buffer != nil && size > 0 {
		for i := 0; i < int(size); i++ {
			rawCall := *(*C.ntg_call_info_struct)(unsafe.Pointer(uintptr(unsafe.Pointer(buffer)) + uintptr(i)*unsafe.Sizeof(C.ntg_call_info_struct{})))
			mapReturn[int64(rawCall.chatId)] = &CallInfo{
				Playback: parseStreamStatus(rawCall.playback),
				Capture:  parseStreamStatus(rawCall.capture),
			}
		}
		defer C.free(unsafe.Pointer(buffer))
	}
	return mapReturn
}

//goland:noinspection GoUnusedExportedFunction
func Version() string {
	var buffer *C.char
	C.ntg_get_version(&buffer)
	defer C.free(unsafe.Pointer(buffer))
	return C.GoString(buffer)
}

func (ctx *Client) Free() {
	C.ntg_destroy(C.uintptr_t(ctx.ptr))
	ctx.handle.Delete()
	C.free(ctx.handlePtr)
}
