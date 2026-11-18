/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
)

func readOggPage(r io.Reader) ([]byte, error) {
	header := make([]byte, 27)
	n, err := io.ReadFull(r, header)
	if err == io.EOF && n == 0 {
		return nil, io.EOF
	}
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(header[:4], []byte("OggS")) || header[4] != 0 {
		return nil, errors.New("invalid OGG page")
	}
	segments := make([]byte, int(header[26]))
	if _, err := io.ReadFull(r, segments); err != nil {
		return nil, err
	}
	size := 0
	for _, segment := range segments {
		size += int(segment)
	}
	page := append(header, segments...)
	page = append(page, make([]byte, size)...)
	_, err = io.ReadFull(r, page[27+len(segments):])
	return page, err
}

func oggCRC(page []byte) uint32 {
	var crc uint32
	for _, b := range page {
		crc ^= uint32(b) << 24
		for range 8 {
			if crc&0x80000000 != 0 {
				crc = crc<<1 ^ 0x04c11db7
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// RepairOGG restores one Spotify stream's page serials and checksums.
func RepairOGG(input, output string) error {
	source, err := os.Open(input)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.CreateTemp(filepath.Dir(output), ".ogg-*")
	if err != nil {
		return err
	}
	defer os.Remove(target.Name())
	defer target.Close()
	pages := 0
	var serial uint32
	for {
		page, err := readOggPage(source)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if pages == 0 {
			serial = binary.LittleEndian.Uint32(page[14:18])
		}
		binary.LittleEndian.PutUint32(page[14:18], serial)
		clear(page[22:26])
		binary.LittleEndian.PutUint32(page[22:26], oggCRC(page))
		if _, err := target.Write(page); err != nil {
			return err
		}
		pages++
	}
	if pages == 0 {
		return errors.New("empty OGG stream")
	}
	if err := target.Close(); err != nil {
		return err
	}
	return os.Rename(target.Name(), output)
}

func oggDuration(r io.Reader) int32 {
	var sampleRate, preSkip, last uint64
	for {
		page, err := readOggPage(r)
		if err != nil {
			break
		}
		body := page[27+int(page[26]):]
		if sampleRate == 0 && len(body) >= 16 && body[0] == 1 && string(body[1:7]) == "vorbis" {
			sampleRate = uint64(binary.LittleEndian.Uint32(body[12:16]))
		} else if sampleRate == 0 && len(body) >= 12 && string(body[:8]) == "OpusHead" {
			sampleRate, preSkip = 48000, uint64(binary.LittleEndian.Uint16(body[10:12]))
		}
		if granule := binary.LittleEndian.Uint64(page[6:14]); granule != ^uint64(0) && granule > last {
			last = granule
		}
	}
	if sampleRate == 0 || last <= preSkip {
		return 0
	}
	return mediaSeconds(last-preSkip, sampleRate)
}
