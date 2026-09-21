package ntgcalls

//#include "ntgcalls.h"
import "C"

// StreamType represents the type of media stream (audio or video).
type StreamType int

// ConnectionMode represents the connection protocol mode.
type ConnectionMode int

// ConnectionKind represents the kind of connection (normal or presentation).
type ConnectionKind int

// ConnectionState represents the state of the network connection.
type ConnectionState int

// StreamStatus represents the activity status of a stream.
type StreamStatus int

// StreamMode represents whether the stream mode is capture or playback.
type StreamMode int

// StreamDevice represents the media device type.
type StreamDevice int

// MediaSource represents the source type of media data.
type MediaSource int

// MediaSegmentQuality represents the quality level of a media segment.
type MediaSegmentQuality int

// MediaSegmentStatus represents the status of a media segment part.
type MediaSegmentStatus int

// StreamEndCallback is invoked when a media stream reaches its end.
type StreamEndCallback func(chatId int64, streamType StreamType, streamDevice StreamDevice)

// UpgradeCallback is invoked when media state is updated.
type UpgradeCallback func(chatId int64, state MediaState)

// ConnectionChangeCallback is invoked when network connection status changes.
type ConnectionChangeCallback func(chatId int64, state NetworkInfo)

// FrameCallback is invoked when media frames are received.
type FrameCallback func(chatId int64, mode StreamMode, device StreamDevice, frames []Frame)

// RemoteSourceCallback is invoked when a remote source state changes.
type RemoteSourceCallback func(chatId int64, source RemoteSource)

// BroadcastTimestampCallback is invoked when a broadcast timestamp is requested.
type BroadcastTimestampCallback func(chatId int64)

// BroadcastPartCallback is invoked when a broadcast segment part is requested.
type BroadcastPartCallback func(chatId int64, segmentPartRequest SegmentPartRequest)

const (
	// MicrophoneStream represents a microphone input stream.
	MicrophoneStream StreamDevice = iota
	// SpeakerStream represents a speaker output stream.
	SpeakerStream
	// CameraStream represents a camera video stream.
	CameraStream
	// ScreenStream represents a screen-sharing video stream.
	ScreenStream
)

const (
	// AudioStream represents an audio stream type.
	AudioStream StreamType = iota
	// VideoStream represents a video stream type.
	VideoStream
)

const (
	// MediaSourceFile represents media from a local file.
	MediaSourceFile MediaSource = 1 << iota
	// MediaSourceShell represents media piped from a shell command.
	MediaSourceShell
	// MediaSourceFFmpeg represents media processed via FFmpeg.
	MediaSourceFFmpeg
	// MediaSourceDevice represents media captured directly from a hardware device.
	MediaSourceDevice
	// MediaSourceDesktop represents media captured from a desktop capture source.
	MediaSourceDesktop
	// MediaSourceExternal represents media provided by an external data stream.
	MediaSourceExternal
)

const (
	// ActiveStream indicates an active media stream.
	ActiveStream StreamStatus = iota
	// PausedStream indicates a paused media stream.
	PausedStream
	// IdlingStream indicates an idling media stream.
	IdlingStream
)

const (
	// RtcConnection indicates WebRTC connection mode.
	RtcConnection ConnectionMode = iota
	// StreamConnection indicates direct stream connection mode.
	StreamConnection
	// RTMPConnection indicates RTMP connection mode.
	RTMPConnection
)

const (
	// Connecting indicates the connection is being established.
	Connecting ConnectionState = iota
	// Connected indicates an active connection.
	Connected
	// Failed indicates a failed connection attempt.
	Failed
	// Timeout indicates a timed out connection attempt.
	Timeout
	// Closed indicates a closed connection.
	Closed
)

const (
	// NormalConnection represents a standard group call or call connection.
	NormalConnection ConnectionKind = iota
	// PresentationConnection represents a screen share or presentation connection.
	PresentationConnection
)

const (
	// CaptureStream represents capture mode for sending media.
	CaptureStream StreamMode = iota
	// PlaybackStream represents playback mode for receiving media.
	PlaybackStream
)

const (
	// SegmentQualityNone indicates no quality setting.
	SegmentQualityNone MediaSegmentQuality = iota - 1
	// SegmentQualityThumbnail indicates thumbnail quality.
	SegmentQualityThumbnail
	// SegmentQualityMedium indicates medium quality.
	SegmentQualityMedium
	// SegmentQualityFull indicates full quality.
	SegmentQualityFull
)

const (
	// SegmentStatusNotReady indicates the segment part is not ready.
	SegmentStatusNotReady MediaSegmentStatus = iota
	// SegmentStatusResyncNeeded indicates resynchronization is required for the segment part.
	SegmentStatusResyncNeeded
	// SegmentStatusSuccess indicates the segment part was successfully downloaded/processed.
	SegmentStatusSuccess
)

// ParseToC converts MediaSource to C.ntg_media_source.
func (ctx MediaSource) ParseToC() C.ntg_media_source {
	switch ctx {
	case MediaSourceFile:
		return C.NTG_MEDIA_SOURCE_FILE
	case MediaSourceShell:
		return C.NTG_MEDIA_SOURCE_SHELL
	case MediaSourceFFmpeg:
		return C.NTG_MEDIA_SOURCE_FFMPEG
	case MediaSourceDevice:
		return C.NTG_MEDIA_SOURCE_DEVICE
	case MediaSourceDesktop:
		return C.NTG_MEDIA_SOURCE_DESKTOP
	case MediaSourceExternal:
		return C.NTG_MEDIA_SOURCE_EXTERNAL
	default:
		return C.NTG_MEDIA_SOURCE_FILE
	}
}

// ParseToC converts StreamMode to C.ntg_stream_mode.
func (ctx StreamMode) ParseToC() C.ntg_stream_mode {
	switch ctx {
	case CaptureStream:
		return C.NTG_STREAM_MODE_CAPTURE
	case PlaybackStream:
		return C.NTG_STREAM_MODE_PLAYBACK
	default:
		return C.NTG_STREAM_MODE_CAPTURE
	}
}

// ParseToC converts StreamDevice to C.ntg_stream_device.
func (ctx StreamDevice) ParseToC() C.ntg_stream_device {
	switch ctx {
	case MicrophoneStream:
		return C.NTG_STREAM_DEVICE_MICROPHONE
	case SpeakerStream:
		return C.NTG_STREAM_DEVICE_SPEAKER
	case CameraStream:
		return C.NTG_STREAM_DEVICE_CAMERA
	case ScreenStream:
		return C.NTG_STREAM_DEVICE_SCREEN
	default:
		return C.NTG_STREAM_DEVICE_MICROPHONE
	}
}

// ParseToC converts MediaSegmentStatus to C.ntg_media_segment_part_status.
func (ctx MediaSegmentStatus) ParseToC() C.ntg_media_segment_part_status {
	switch ctx {
	case SegmentStatusNotReady:
		return C.NTG_MEDIA_SEGMENT_PART_STATUS_NOT_READY
	case SegmentStatusResyncNeeded:
		return C.NTG_MEDIA_SEGMENT_PART_STATUS_RESYNC_NEEDED
	case SegmentStatusSuccess:
		return C.NTG_MEDIA_SEGMENT_PART_STATUS_SUCCESS
	default:
		return C.NTG_MEDIA_SEGMENT_PART_STATUS_NOT_READY
	}
}
