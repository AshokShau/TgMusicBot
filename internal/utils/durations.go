/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package utils

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/AshokShau/gotdbot/logger"
)

// GetMediaDuration reads local media in Go and probes remote URLs with ffprobe.
var log = logger.New()

func GetMediaDuration(input string) int32 {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()

		output, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", input).Output()
		if err != nil {
			log.Warnf("ffprobe failed to read remote duration: %v", err)
			return 0
		}
		seconds, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
		if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 || math.Round(seconds) > math.MaxInt32 {
			return 0
		}
		return int32(math.Round(seconds))
	}

	f, err := os.Open(input)
	if err != nil {
		return 0
	}
	defer f.Close()
	header := make([]byte, 12)
	if _, err := io.ReadFull(f, header); err != nil {
		return 0
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0
	}
	switch {
	case string(header[:4]) == "OggS":
		return oggDuration(f)
	case string(header[:4]) == "fLaC":
		return flacDuration(f)
	case string(header[:4]) == "RIFF" && string(header[8:12]) == "WAVE":
		return wavDuration(f)
	case string(header[4:8]) == "ftyp":
		return mp4Duration(f)
	case string(header[:3]) == "ID3" || header[0] == 0xff:
		return mp3Duration(f)
	case binary.BigEndian.Uint32(header[:4]) == 0x1a45dfa3:
		return webmDuration(f)
	}
	return 0
}

func mediaSeconds(samples, rate uint64) int32 {
	if rate == 0 || samples/rate >= math.MaxInt32 {
		return 0
	}
	seconds := samples / rate
	if samples%rate >= (rate+1)/2 {
		seconds++
	}
	return int32(seconds)
}

func flacDuration(f *os.File) int32 {
	data := make([]byte, 42)
	if _, err := io.ReadFull(f, data); err != nil || data[4]&0x7f != 0 || binary.BigEndian.Uint32(data[4:8])&0xffffff != 34 {
		return 0
	}
	info := data[8:]
	rate := uint64(info[10])<<12 | uint64(info[11])<<4 | uint64(info[12]>>4)
	samples := uint64(info[13]&0x0f)<<32 | uint64(binary.BigEndian.Uint32(info[14:18]))
	return mediaSeconds(samples, rate)
}

func wavDuration(f *os.File) int32 {
	if _, err := f.Seek(12, io.SeekStart); err != nil {
		return 0
	}
	var byteRate, dataSize uint64
	for {
		header := make([]byte, 8)
		if _, err := io.ReadFull(f, header); err != nil {
			break
		}
		size := uint64(binary.LittleEndian.Uint32(header[4:8]))
		if string(header[:4]) == "fmt " && size >= 12 {
			format := make([]byte, 12)
			if _, err := io.ReadFull(f, format); err != nil {
				break
			}
			byteRate = uint64(binary.LittleEndian.Uint32(format[8:12]))
			if _, err := f.Seek(int64(size-12+size%2), io.SeekCurrent); err != nil {
				break
			}
		} else {
			if string(header[:4]) == "data" {
				dataSize = size
				break
			}
			if _, err := f.Seek(int64(size+size%2), io.SeekCurrent); err != nil {
				break
			}
		}
	}
	return mediaSeconds(dataSize, byteRate)
}

func mp4Box(f *os.File, start, end int64, wanted string) (int64, int64) {
	for pos := start; pos+8 <= end; {
		header := make([]byte, 16)
		if _, err := f.ReadAt(header[:8], pos); err != nil {
			break
		}
		size, offset := int64(binary.BigEndian.Uint32(header[:4])), int64(8)
		if size == 1 {
			if _, err := f.ReadAt(header[8:16], pos+8); err != nil {
				break
			}
			size, offset = int64(binary.BigEndian.Uint64(header[8:16])), 16
		} else if size == 0 {
			size = end - pos
		}
		if size < offset || pos+size > end {
			break
		}
		if string(header[4:8]) == wanted {
			return pos + offset, pos + size
		}
		pos += size
	}
	return 0, 0
}

func mp4Duration(f *os.File) int32 {
	info, err := f.Stat()
	if err != nil {
		return 0
	}
	start, end := mp4Box(f, 0, info.Size(), "moov")
	if start == 0 {
		return 0
	}
	start, end = mp4Box(f, start, end, "mvhd")
	if start == 0 || end-start < 20 {
		return 0
	}
	data := make([]byte, 32)
	if _, err := f.ReadAt(data[:min(int64(len(data)), end-start)], start); err != nil {
		return 0
	}
	if data[0] == 0 {
		return mediaSeconds(uint64(binary.BigEndian.Uint32(data[16:20])), uint64(binary.BigEndian.Uint32(data[12:16])))
	}
	if data[0] == 1 && end-start >= 32 {
		return mediaSeconds(binary.BigEndian.Uint64(data[24:32]), uint64(binary.BigEndian.Uint32(data[20:24])))
	}
	return 0
}

func mp3Frame(header uint32) (int64, uint64) {
	version, layer := (header>>19)&3, (header>>17)&3
	bitrateIndex, sampleIndex := (header>>12)&15, (header>>10)&3
	if header>>21 != 0x7ff || version == 1 || layer != 1 || bitrateIndex == 0 || bitrateIndex == 15 || sampleIndex == 3 {
		return 0, 0
	}
	bitrates := [...]uint32{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320}
	if version != 3 {
		bitrates = [...]uint32{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160}
	}
	rate := [...]uint32{44100, 48000, 32000}[sampleIndex]
	if version == 2 {
		rate /= 2
	} else if version == 0 {
		rate /= 4
	}
	coefficient, samples := uint32(144000), uint64(1152)
	if version != 3 {
		coefficient, samples = 72000, 576
	}
	return int64(coefficient*bitrates[bitrateIndex]/rate + (header>>9)&1), samples * 1000000 / uint64(rate)
}

func mp3Duration(f *os.File) int32 {
	info, err := f.Stat()
	if err != nil {
		return 0
	}
	var pos int64
	header := make([]byte, 10)
	if _, err := f.ReadAt(header, 0); err == nil && string(header[:3]) == "ID3" {
		pos = 10 + int64(header[6]&0x7f)<<21 + int64(header[7]&0x7f)<<14 + int64(header[8]&0x7f)<<7 + int64(header[9]&0x7f)
	}
	var micros uint64
	for misses := 0; pos+4 <= info.Size() && misses < 65536; {
		if _, err := f.ReadAt(header[:4], pos); err != nil {
			break
		}
		length, frameMicros := mp3Frame(binary.BigEndian.Uint32(header[:4]))
		if length == 0 || pos+length > info.Size() {
			pos++
			misses++
			continue
		}
		micros += frameMicros
		pos += length
		misses = 0
	}
	return mediaSeconds(micros, 1000000)
}

func ebmlNumber(f *os.File, pos int64, size bool) (uint64, int64) {
	var first [1]byte
	if _, err := f.ReadAt(first[:], pos); err != nil || first[0] == 0 {
		return 0, 0
	}
	length := int64(1)
	for first[0]&(0x80>>uint(length-1)) == 0 {
		length++
	}
	if length > 8 {
		return 0, 0
	}
	value := uint64(first[0])
	if size {
		value &= uint64(0xff >> uint(length))
	}
	for i := int64(1); i < length; i++ {
		if _, err := f.ReadAt(first[:], pos+i); err != nil {
			return 0, 0
		}
		value = value<<8 | uint64(first[0])
	}
	return value, length
}

func ebmlElement(f *os.File, pos int64) (uint64, int64, int64) {
	id, idSize := ebmlNumber(f, pos, false)
	if idSize == 0 {
		return 0, 0, 0
	}
	size, sizeSize := ebmlNumber(f, pos+idSize, true)
	if sizeSize == 0 {
		return 0, 0, 0
	}
	return id, pos + idSize + sizeSize, int64(size)
}

func webmDuration(f *os.File) int32 {
	info, err := f.Stat()
	if err != nil {
		return 0
	}
	var segment, limit int64
	for pos := int64(0); pos < info.Size(); {
		id, body, size := ebmlElement(f, pos)
		if body == 0 {
			return 0
		}
		if id == 0x18538067 {
			segment, limit = body, info.Size()
			if size < limit-body {
				limit = body + size
			}
			break
		}
		pos = body + size
	}
	if segment == 0 {
		return 0
	}
	for pos := segment; pos < limit; {
		id, body, size := ebmlElement(f, pos)
		if body == 0 || body+size > limit {
			return 0
		}
		if id == 0x1549a966 {
			return webmInfoDuration(f, body, body+size)
		}
		pos = body + size
	}
	return 0
}

func webmInfoDuration(f *os.File, start, end int64) int32 {
	scale := uint64(1000000)
	var duration float64
	for pos := start; pos < end; {
		id, body, size := ebmlElement(f, pos)
		if body == 0 || body+size > end {
			break
		}
		if size == 4 || size == 8 {
			var data [8]byte
			if _, err := f.ReadAt(data[:size], body); err != nil {
				break
			}
			if id == 0x2ad7b1 {
				scale = binary.BigEndian.Uint64(data[:]) >> uint(8*(8-size))
			}
			if id == 0x4489 && size == 4 {
				duration = float64(math.Float32frombits(binary.BigEndian.Uint32(data[:4])))
			}
			if id == 0x4489 && size == 8 {
				duration = math.Float64frombits(binary.BigEndian.Uint64(data[:]))
			}
		} else if id == 0x2ad7b1 && size > 0 && size < 4 {
			data := make([]byte, size)
			if _, err := f.ReadAt(data, body); err != nil {
				break
			}
			scale = 0
			for _, b := range data {
				scale = scale<<8 | uint64(b)
			}
		}
		pos = body + size
	}
	seconds := duration * float64(scale) / 1e9
	if seconds < 0 || seconds > math.MaxInt32 || math.IsNaN(seconds) {
		return 0
	}
	return int32(math.Round(seconds))
}

func SecToMin(seconds int32) string {
	if seconds <= 0 {
		return "0:00"
	}

	d := seconds / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	if d > 0 {
		return fmt.Sprintf("%dd %02d:%02d:%02d", d, h, m, s)
	}

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}

	return fmt.Sprintf("%d:%02d", m, s)
}
