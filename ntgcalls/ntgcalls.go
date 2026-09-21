package ntgcalls

/*
#include "ntgcalls.h"
#include <stdlib.h>

extern void handleStreamEnd(ntg_instance* handle, int64_t chatID, ntg_stream_type streamType, ntg_stream_device streamDevice, void* user_data);
extern void handleUpgrade(ntg_instance* handle, int64_t chatID, ntg_media_state state, void* user_data);
extern void handleConnectionChange(ntg_instance* handle, int64_t chatID, ntg_connection_info connectionInfo, void* user_data);
extern void handleFrames(ntg_instance* handle, int64_t chatID, ntg_stream_mode streamMode, ntg_stream_device streamDevice, ntg_frame* frames, size_t size, void* user_data);
extern void handleRemoteSourceChange(ntg_instance* handle, int64_t chatID, ntg_remote_source remoteSource, void* user_data);
extern void handleRequestBroadcastTimestamp(ntg_instance* handle, int64_t chatID, void* user_data);
extern void handleRequestBroadcastPart(ntg_instance* handle, int64_t chatID, ntg_segment_part_request segmentPartRequest, void* user_data);
extern void handleLogs(ntg_log_message logMessage, void* user_data);
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func init() {
	C.ntg_set_log_callback((C.ntg_log_cb)(unsafe.Pointer(C.handleLogs)), nil)
}

var (
	loggerNTGCalls = NewLogger("ntgcalls", LevelDebug)
	loggerWebRTC   = NewLogger("webrtc", LevelFatal)
)

// NTgCalls creates and initializes a new Client instance.
func NTgCalls() *Client {
	instance := &Client{
		handle: C.ntg_instance_create(),
	}
	selfPointer := unsafe.Pointer(instance)
	C.ntg_on_stream_end_callback(instance.handle, (C.ntg_stream_end_callback_cb)(unsafe.Pointer(C.handleStreamEnd)), selfPointer)
	C.ntg_on_upgrade_callback(instance.handle, (C.ntg_upgrade_callback_cb)(unsafe.Pointer(C.handleUpgrade)), selfPointer)
	C.ntg_on_connection_change_callback(instance.handle, (C.ntg_connection_change_callback_cb)(unsafe.Pointer(C.handleConnectionChange)), selfPointer)
	C.ntg_on_frames_callback(instance.handle, (C.ntg_frames_callback_cb)(unsafe.Pointer(C.handleFrames)), selfPointer)
	C.ntg_on_remote_source_change_callback(instance.handle, (C.ntg_remote_source_change_callback_cb)(unsafe.Pointer(C.handleRemoteSourceChange)), selfPointer)
	C.ntg_on_request_broadcast_timestamp_callback(instance.handle, (C.ntg_request_broadcast_timestamp_callback_cb)(unsafe.Pointer(C.handleRequestBroadcastTimestamp)), selfPointer)
	C.ntg_on_request_broadcast_part_callback(instance.handle, (C.ntg_request_broadcast_part_callback_cb)(unsafe.Pointer(C.handleRequestBroadcastPart)), selfPointer)
	return instance
}

//export handleLogs
func handleLogs(logMessage C.ntg_log_message, _ unsafe.Pointer) {
	message := fmt.Sprintf(
		"(%v:%v) %v",
		C.GoString(logMessage.file),
		uint32(logMessage.line),
		C.GoString(logMessage.message),
	)

	var lg *Logger
	if logMessage.source == C.NTG_LOG_SOURCE_WEBRTC {
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
func handleStreamEnd(_ *C.ntg_instance, chatID C.int64_t, streamType C.ntg_stream_type, streamDevice C.ntg_stream_device, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	var goStreamType StreamType
	if streamType == C.NTG_STREAM_TYPE_AUDIO {
		goStreamType = AudioStream
	} else {
		goStreamType = VideoStream
	}
	for _, x0 := range self.streamEndCallbacks {
		go x0(goChatID, goStreamType, parseStreamDevice(streamDevice))
	}
}

//export handleUpgrade
func handleUpgrade(_ *C.ntg_instance, chatID C.int64_t, state C.ntg_media_state, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	goState := MediaState{
		Muted:              bool(state.muted),
		VideoPaused:        bool(state.video_paused),
		VideoStopped:       bool(state.video_stopped),
		PresentationPaused: bool(state.presentation_paused),
	}
	for _, x0 := range self.upgradeCallbacks {
		go x0(goChatID, goState)
	}
}

//export handleConnectionChange
func handleConnectionChange(_ *C.ntg_instance, chatID C.int64_t, connectionInfo C.ntg_connection_info, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	var goCallState NetworkInfo
	switch connectionInfo.kind {
	case C.NTG_CONNECTION_KIND_NORMAL:
		goCallState.Kind = NormalConnection
	case C.NTG_CONNECTION_KIND_PRESENTATION:
		goCallState.Kind = PresentationConnection
	}
	goCallState.State = parseConnectionState(connectionInfo.state)
	for _, x0 := range self.connectionChangeCallbacks {
		go x0(goChatID, goCallState)
	}
}

//export handleFrames
func handleFrames(_ *C.ntg_instance, chatID C.int64_t, streamMode C.ntg_stream_mode, streamDevice C.ntg_stream_device, frames *C.ntg_frame, size C.size_t, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	var goStreamMode StreamMode
	switch streamMode {
	case C.NTG_STREAM_MODE_CAPTURE:
		goStreamMode = CaptureStream
	case C.NTG_STREAM_MODE_PLAYBACK:
		goStreamMode = PlaybackStream
	}
	rawFrames := make([]Frame, size)
	for i := uint64(0); i < uint64(size); i++ {
		rawFrame := *(*C.ntg_frame)(unsafe.Pointer(uintptr(unsafe.Pointer(frames)) + uintptr(i)*unsafe.Sizeof(C.ntg_frame{})))
		rawFrames[i] = Frame{
			Ssrc: uint32(rawFrame.ssrc),
			Data: C.GoBytes(unsafe.Pointer(rawFrame.data), C.int(rawFrame.data_len)),
			FrameData: FrameData{
				AbsoluteCaptureTimestampMs: int64(rawFrame.frame_data.absolute_capture_timestamp_ms),
				Width:                      uint16(rawFrame.frame_data.width),
				Height:                     uint16(rawFrame.frame_data.height),
				Rotation:                   uint16(rawFrame.frame_data.rotation),
			},
		}
	}
	for _, x0 := range self.frameCallbacks {
		go x0(goChatID, goStreamMode, parseStreamDevice(streamDevice), rawFrames)
	}
}

//export handleRemoteSourceChange
func handleRemoteSourceChange(_ *C.ntg_instance, chatID C.int64_t, remoteSource C.ntg_remote_source, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	goRemoteSource := RemoteSource{
		Ssrc:   uint32(remoteSource.ssrc),
		State:  parseStreamStatus(remoteSource.state),
		Device: parseStreamDevice(remoteSource.device),
	}
	for _, x0 := range self.remoteSourceCallbacks {
		go x0(goChatID, goRemoteSource)
	}
}

//export handleRequestBroadcastTimestamp
func handleRequestBroadcastTimestamp(_ *C.ntg_instance, chatID C.int64_t, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
	goChatID := int64(chatID)
	for _, x0 := range self.broadcastTimestampCallbacks {
		go x0(goChatID)
	}
}

//export handleRequestBroadcastPart
func handleRequestBroadcastPart(_ *C.ntg_instance, chatID C.int64_t, segmentPartRequest C.ntg_segment_part_request, ptr unsafe.Pointer) {
	self := (*Client)(ptr)
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
		SegmentID:     int64(segmentPartRequest.segment_id),
		PartID:        int32(segmentPartRequest.part_id),
		Limit:         int32(segmentPartRequest.limit),
		Timestamp:     int64(segmentPartRequest.timestamp),
		QualityUpdate: bool(segmentPartRequest.quality_update),
		ChannelID:     int32(segmentPartRequest.channel_id),
		Quality:       goSegmentQuality,
	}
	for _, x0 := range self.broadcastPartCallbacks {
		go x0(goChatID, goSegmentPartRequest)
	}
}

// OnStreamEnd registers a callback for stream end events.
func (ctx *Client) OnStreamEnd(callback StreamEndCallback) {
	ctx.streamEndCallbacks = append(ctx.streamEndCallbacks, callback)
}

// OnUpgrade registers a callback for media state upgrade events.
func (ctx *Client) OnUpgrade(callback UpgradeCallback) {
	ctx.upgradeCallbacks = append(ctx.upgradeCallbacks, callback)
}

// OnConnectionChange registers a callback for network connection state changes.
func (ctx *Client) OnConnectionChange(callback ConnectionChangeCallback) {
	ctx.connectionChangeCallbacks = append(ctx.connectionChangeCallbacks, callback)
}

// OnFrame registers a callback for receiving media frames.
func (ctx *Client) OnFrame(callback FrameCallback) {
	ctx.frameCallbacks = append(ctx.frameCallbacks, callback)
}

// OnRemoteSourceChange registers a callback for remote source changes.
func (ctx *Client) OnRemoteSourceChange(callback RemoteSourceCallback) {
	ctx.remoteSourceCallbacks = append(ctx.remoteSourceCallbacks, callback)
}

// OnRequestBroadcastTimestamp registers a callback for broadcast timestamp requests.
func (ctx *Client) OnRequestBroadcastTimestamp(callback BroadcastTimestampCallback) {
	ctx.broadcastTimestampCallbacks = append(ctx.broadcastTimestampCallbacks, callback)
}

// OnRequestBroadcastPart registers a callback for broadcast part requests.
func (ctx *Client) OnRequestBroadcastPart(callback BroadcastPartCallback) {
	ctx.broadcastPartCallbacks = append(ctx.broadcastPartCallbacks, callback)
}

// GetState retrieves the current media state for the specified chat ID.
func (ctx *Client) GetState(chatId int64) (MediaState, error) {
	var buffer C.ntg_media_state
	res := C.ntg_get_state(ctx.handle, C.int64_t(chatId), &buffer)
	err := parseResult(res)
	if err != nil {
		return MediaState{}, err
	}
	return MediaState{
		Muted:              bool(buffer.muted),
		VideoPaused:        bool(buffer.video_paused),
		VideoStopped:       bool(buffer.video_stopped),
		PresentationPaused: bool(buffer.presentation_paused),
	}, nil
}

// GetConnectionMode retrieves the connection mode for the specified chat ID.
func (ctx *Client) GetConnectionMode(chatId int64) (ConnectionMode, error) {
	var buffer C.ntg_connection_mode
	res := C.ntg_get_connection_mode(ctx.handle, C.int64_t(chatId), &buffer)
	err := parseResult(res)
	if err != nil {
		return ConnectionMode(0), err
	}
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

// CreateCall initializes a group call for the specified chat ID and returns the connection payload.
func (ctx *Client) CreateCall(chatId int64) (string, error) {
	var buffer *C.char
	res := C.ntg_create_call(ctx.handle, C.int64_t(chatId), &buffer)
	if err := parseResult(res); err != nil {
		return "", err
	}
	defer C.ntg_string_free(buffer)
	return C.GoString(buffer), nil
}

// InitPresentation initializes a presentation/screen share for the specified chat ID and returns the payload.
func (ctx *Client) InitPresentation(chatId int64) (string, error) {
	var buffer *C.char
	res := C.ntg_init_presentation(ctx.handle, C.int64_t(chatId), &buffer)
	if err := parseResult(res); err != nil {
		return "", err
	}
	defer C.ntg_string_free(buffer)
	return C.GoString(buffer), nil
}

// StopPresentation stops an active presentation/screen share for the specified chat ID.
func (ctx *Client) StopPresentation(chatId int64) error {
	res := C.ntg_stop_presentation(ctx.handle, C.int64_t(chatId))
	return parseResult(res)
}

// AddIncomingVideo registers an incoming video source from an endpoint.
func (ctx *Client) AddIncomingVideo(chatId int64, userId int64, endpoint string, ssrcGroups []SsrcGroup) (uint32, error) {
	cEndpoint := C.CString(endpoint)
	defer C.free(unsafe.Pointer(cEndpoint))
	rawGroups, lenGroups := parseSsrcGroups(ssrcGroups)
	defer freeSsrcGroups(rawGroups, lenGroups)
	var out C.uint32_t
	res := C.ntg_add_incoming_video(ctx.handle, C.int64_t(chatId), C.int64_t(userId), cEndpoint, rawGroups, lenGroups, &out)
	return uint32(out), parseResult(res)
}

// RemoveIncomingVideo removes an incoming video source for the specified endpoint.
func (ctx *Client) RemoveIncomingVideo(chatId int64, endpoint string) error {
	cEndpoint := C.CString(endpoint)
	defer C.free(unsafe.Pointer(cEndpoint))
	var out C.bool
	res := C.ntg_remove_incoming_video(ctx.handle, C.int64_t(chatId), cEndpoint, &out)
	return parseResult(res)
}

// GetProtocol retrieves WebRTC/protocol details supported by ntgcalls.
func GetProtocol() Protocol {
	var buffer C.ntg_protocol
	res := C.ntg_get_protocol(&buffer)
	if res != C.NTG_OK {
		return Protocol{}
	}
	defer C.ntg_protocol_free(&buffer)
	return Protocol{
		MinLayer:     int32(buffer.min_layer),
		MaxLayer:     int32(buffer.max_layer),
		UdpReflector: bool(buffer.udp_reflector),
		Versions:     parseStringVector(buffer.library_versions, buffer.library_versions_len),
	}
}

// Connect connects to the group call / voice chat using the provided parameters.
func (ctx *Client) Connect(chatId int64, params string, isPresentation bool) error {
	cParams := C.CString(params)
	defer C.free(unsafe.Pointer(cParams))
	res := C.ntg_connect(ctx.handle, C.int64_t(chatId), cParams, C.bool(isPresentation))
	return parseResult(res)
}

// SetStreamSources updates the media description for capture or playback mode.
func (ctx *Client) SetStreamSources(chatId int64, streamMode StreamMode, desc MediaDescription) error {
	cDesc := desc.ParseToC()
	res := C.ntg_set_stream_sources(ctx.handle, C.int64_t(chatId), streamMode.ParseToC(), cDesc)
	return parseResult(res)
}

// SendExternalFrame sends an external media frame for a specified device.
func (ctx *Client) SendExternalFrame(chatId int64, streamDevice StreamDevice, data []byte, frameData FrameData) error {
	dataC, dataSize := parseBytes(data)
	if dataC != nil {
		defer C.free(unsafe.Pointer(dataC))
	}
	res := C.ntg_send_external_frame(ctx.handle, C.int64_t(chatId), streamDevice.ParseToC(), dataC, dataSize, frameData.ParseToC())
	return parseResult(res)
}

// SendBroadcastTimestamp sends a broadcast timestamp to the specified chat ID.
func (ctx *Client) SendBroadcastTimestamp(chatId int64, timestamp int64) error {
	res := C.ntg_send_broadcast_timestamp(ctx.handle, C.int64_t(chatId), C.int64_t(timestamp))
	return parseResult(res)
}

// SendBroadcastPart sends a broadcast segment part for streaming.
func (ctx *Client) SendBroadcastPart(chatId int64, segmentID int64, partID int32, status MediaSegmentStatus, qualityUpdate bool, data []byte) error {
	dataC, dataSize := parseBytes(data)
	if dataC != nil {
		defer C.free(unsafe.Pointer(dataC))
	}
	res := C.ntg_send_broadcast_part(ctx.handle, C.int64_t(chatId), C.int64_t(segmentID), C.int32_t(partID), status.ParseToC(), C.bool(qualityUpdate), dataC, dataSize)
	return parseResult(res)
}

// Pause pauses streaming for the given chat ID.
func (ctx *Client) Pause(chatId int64) (bool, error) {
	var out C.bool
	res := C.ntg_pause(ctx.handle, C.int64_t(chatId), &out)
	return bool(out), parseResult(res)
}

// Resume resumes streaming for the given chat ID.
func (ctx *Client) Resume(chatId int64) (bool, error) {
	var out C.bool
	res := C.ntg_resume(ctx.handle, C.int64_t(chatId), &out)
	return bool(out), parseResult(res)
}

// Mute mutes audio for the given chat ID.
func (ctx *Client) Mute(chatId int64) (bool, error) {
	var out C.bool
	res := C.ntg_mute(ctx.handle, C.int64_t(chatId), &out)
	return bool(out), parseResult(res)
}

// UnMute unmutes audio for the given chat ID.
func (ctx *Client) UnMute(chatId int64) (bool, error) {
	var out C.bool
	res := C.ntg_unmute(ctx.handle, C.int64_t(chatId), &out)
	return bool(out), parseResult(res)
}

// Stop stops the call for the specified chat ID.
func (ctx *Client) Stop(chatId int64) error {
	res := C.ntg_stop(ctx.handle, C.int64_t(chatId))
	return parseResult(res)
}

// Time returns the playback or capture stream time in milliseconds.
func (ctx *Client) Time(chatId int64, streamMode StreamMode) (uint64, error) {
	var out C.uint64_t
	res := C.ntg_time(ctx.handle, C.int64_t(chatId), streamMode.ParseToC(), &out)
	return uint64(out), parseResult(res)
}

// GetMediaDevices returns available hardware audio and video devices.
func GetMediaDevices() MediaDevices {
	var buffer C.ntg_media_devices
	res := C.ntg_get_media_devices(&buffer)
	if res != C.NTG_OK {
		return MediaDevices{}
	}
	defer C.ntg_media_devices_free(&buffer)
	return MediaDevices{
		Microphone: parseDeviceInfoVector(buffer.microphone, buffer.microphone_len),
		Speaker:    parseDeviceInfoVector(buffer.speaker, buffer.speaker_len),
		Camera:     parseDeviceInfoVector(buffer.camera, buffer.camera_len),
		Screen:     parseDeviceInfoVector(buffer.screen, buffer.screen_len),
	}
}

// CpuUsage retrieves the current CPU usage of the ntgcalls instance.
func (ctx *Client) CpuUsage() (float64, error) {
	var buffer C.double
	res := C.ntg_cpu_usage(ctx.handle, &buffer)
	return float64(buffer), parseResult(res)
}

// EnableGLibLoop enables or disables the GLib main event loop integration.
func (ctx *Client) EnableGLibLoop(enable bool) {
	C.ntg_enable_glib_loop(C.bool(enable))
}

// Calls returns a map of chat IDs to CallInfo for all active calls.
func (ctx *Client) Calls() map[int64]*CallInfo {
	mapReturn := make(map[int64]*CallInfo)
	var buffer *C.ntg_call_info_entry
	var size C.size_t
	res := C.ntg_calls(ctx.handle, &buffer, &size)
	if res != C.NTG_OK || buffer == nil || size == 0 {
		return mapReturn
	}
	defer C.ntg_call_info_entry_free(buffer, size)
	for i := 0; i < int(size); i++ {
		rawCall := *(*C.ntg_call_info_entry)(unsafe.Pointer(uintptr(unsafe.Pointer(buffer)) + uintptr(i)*unsafe.Sizeof(C.ntg_call_info_entry{})))
		mapReturn[int64(rawCall.key)] = &CallInfo{
			Playback: parseStreamStatus(rawCall.value.playback),
			Capture:  parseStreamStatus(rawCall.value.capture),
		}
	}
	return mapReturn
}

// Version returns the version string of the ntgcalls C library.
func Version() string {
	version := C.ntg_get_version()
	if version != nil {
		return C.GoString(version)
	}
	return ""
}

// Free destroys the underlying C ntgcalls instance.
func (ctx *Client) Free() {
	if ctx.handle != nil {
		C.ntg_instance_destroy(ctx.handle)
		ctx.handle = nil
	}
}
