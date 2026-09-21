package ntgcalls

//#include "ntgcalls.h"
//#include <stdlib.h>
import "C"
import (
	"fmt"
	"unsafe"
)

func parseConnectionState(state C.ntg_connection_state) ConnectionState {
	switch state {
	case C.NTG_CONNECTION_STATE_CONNECTING:
		return Connecting
	case C.NTG_CONNECTION_STATE_CONNECTED:
		return Connected
	case C.NTG_CONNECTION_STATE_FAILED:
		return Failed
	case C.NTG_CONNECTION_STATE_TIMEOUT:
		return Timeout
	case C.NTG_CONNECTION_STATE_CLOSED:
		return Closed
	}
	return Connecting
}

func parseStreamDevice(device C.ntg_stream_device) StreamDevice {
	var goDevice StreamDevice
	switch device {
	case C.NTG_STREAM_DEVICE_MICROPHONE:
		goDevice = MicrophoneStream
	case C.NTG_STREAM_DEVICE_SPEAKER:
		goDevice = SpeakerStream
	case C.NTG_STREAM_DEVICE_CAMERA:
		goDevice = CameraStream
	case C.NTG_STREAM_DEVICE_SCREEN:
		goDevice = ScreenStream
	}
	return goDevice
}

func parseStreamStatus(status C.ntg_stream_status) StreamStatus {
	switch status {
	case C.NTG_STREAM_STATUS_ACTIVE:
		return ActiveStream
	case C.NTG_STREAM_STATUS_PAUSED:
		return PausedStream
	case C.NTG_STREAM_STATUS_IDLING:
		return IdlingStream
	}
	return ActiveStream
}

func parseResult(res C.ntg_result) error {
	if res != C.NTG_OK {
		lastErr := C.ntg_last_error()
		if lastErr != nil {
			return fmt.Errorf("%s", C.GoString(lastErr))
		}
		return fmt.Errorf("ntgcalls error code: %d", int(res))
	}
	return nil
}

func parseBytes(data []byte) (*C.uint8_t, C.size_t) {
	if len(data) > 0 {
		rawBytes := C.CBytes(data)
		return (*C.uint8_t)(rawBytes), C.size_t(len(data))
	}
	return nil, 0
}

func parseStringVector(data **C.char, size C.size_t) []string {
	if data == nil || size == 0 {
		return nil
	}
	result := make([]string, size)
	for i := 0; i < int(size); i++ {
		pointer := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(data)) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		if pointer != nil {
			result[i] = C.GoString(pointer)
		}
	}
	return result
}

func parseUint32VectorC(data []uint32) (*C.uint32_t, C.size_t) {
	if len(data) > 0 {
		cData := C.malloc(C.size_t(len(data)) * C.size_t(unsafe.Sizeof(C.uint32_t(0))))
		if cData == nil {
			return nil, 0
		}
		ssrcs := (*C.uint32_t)(cData)
		for i, v := range data {
			*(*C.uint32_t)(unsafe.Pointer(uintptr(unsafe.Pointer(ssrcs)) + uintptr(i)*unsafe.Sizeof(C.uint32_t(0)))) = C.uint32_t(v)
		}
		return ssrcs, C.size_t(len(data))
	}
	return nil, 0
}

func parseSsrcGroups(ssrcGroups []SsrcGroup) (*C.ntg_ssrc_group, C.size_t) {
	if len(ssrcGroups) > 0 {
		cData := C.malloc(C.size_t(len(ssrcGroups)) * C.size_t(unsafe.Sizeof(C.ntg_ssrc_group{})))
		if cData == nil {
			return nil, 0
		}
		rawGroups := (*C.ntg_ssrc_group)(cData)
		for i, group := range ssrcGroups {
			ssrcsC, sizeSsrcs := parseUint32VectorC(group.Ssrcs)
			groupPtr := (*C.ntg_ssrc_group)(unsafe.Pointer(uintptr(unsafe.Pointer(rawGroups)) + uintptr(i)*unsafe.Sizeof(C.ntg_ssrc_group{})))
			groupPtr.semantics = C.CString(group.Semantics)
			groupPtr.ssrcs = ssrcsC
			groupPtr.ssrcs_len = sizeSsrcs
		}
		return rawGroups, C.size_t(len(ssrcGroups))
	}
	return nil, 0
}

func freeSsrcGroups(groups *C.ntg_ssrc_group, size C.size_t) {
	if groups == nil {
		return
	}
	for i := 0; i < int(size); i++ {
		groupPtr := (*C.ntg_ssrc_group)(unsafe.Pointer(uintptr(unsafe.Pointer(groups)) + uintptr(i)*unsafe.Sizeof(C.ntg_ssrc_group{})))
		if groupPtr.semantics != nil {
			C.free(unsafe.Pointer(groupPtr.semantics))
		}
		if groupPtr.ssrcs != nil {
			C.free(unsafe.Pointer(groupPtr.ssrcs))
		}
	}
	C.free(unsafe.Pointer(groups))
}

func parseDeviceInfoVector(devices *C.ntg_device_info, size C.size_t) []DeviceInfo {
	if devices == nil || size == 0 {
		return nil
	}
	rawDevices := make([]DeviceInfo, size)
	for i := 0; i < int(size); i++ {
		device := *(*C.ntg_device_info)(unsafe.Pointer(uintptr(unsafe.Pointer(devices)) + uintptr(i)*unsafe.Sizeof(C.ntg_device_info{})))
		rawDevices[i] = DeviceInfo{
			Name:     C.GoString(device.name),
			Metadata: C.GoString(device.metadata),
		}
	}
	return rawDevices
}
