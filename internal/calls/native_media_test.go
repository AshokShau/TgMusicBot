package calls

import (
	"ashokshau/tgmusic/ntgcalls"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalWAVUsesNativeReader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.wav")
	// One second of mono 48 kHz PCM silence in a valid WAV container.
	data := make([]byte, 44+48000*2)
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], 48000)
	binary.LittleEndian.PutUint32(data[28:], 48000*2)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(len(data)-44))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	desc := getMediaDescription(path, false, 0, "")
	if desc.Microphone == nil || desc.Microphone.MediaSource != ntgcalls.MediaSourceFile ||
		desc.Microphone.Format == nil || *desc.Microphone.Format != ntgcalls.MediaFormatAuto {
		t.Fatalf("local WAV was not routed through NTgCalls: %+v", desc.Microphone)
	}
}

func TestNativeMediaDescriptionRequiresSupportedStreams(t *testing.T) {
	info := ntgcalls.MediaInfo{
		Audio: &ntgcalls.AudioInfo{Supported: true, Passthrough: true, Channels: 1},
		Video: &ntgcalls.VideoInfo{Supported: false},
	}
	if _, ok := nativeMediaDescription("clip.mp4", true, info); ok {
		t.Fatal("unsupported video was accepted")
	}
	if desc, ok := nativeMediaDescription("clip.mp4", false, info); !ok || desc.Camera != nil || desc.Microphone.ChannelCount != 1 {
		t.Fatal("supported audio-only file was rejected")
	}
}
