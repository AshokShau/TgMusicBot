package ntgcalls

// MediaDevices contains available media input and output devices.
type MediaDevices struct {
	Microphone []DeviceInfo
	Speaker    []DeviceInfo
	Camera     []DeviceInfo
	Screen     []DeviceInfo
}
