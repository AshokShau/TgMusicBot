/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// GetFileDur extracts the duration of a media file from a Telegram message.
func GetFileDur(m *tg.NewMessage) int {
	if !m.IsMedia() {
		return 0
	}

	switch media := m.Media().(type) {
	case *tg.MessageMediaDocument:
		return getDocumentDuration(media)
	case *tg.MessageMediaPhoto:
		return 0
	default:
		m.Client.Logger.Info("Unsupported media type: %T", media)
		return 0
	}
}

// getDocumentDuration extracts the duration from a document's attributes.
func getDocumentDuration(media *tg.MessageMediaDocument) int {
	doc, ok := media.Document.(*tg.DocumentObj)
	if !ok {
		log.Printf("Unsupported document type: %T", media.Document)
		return 0
	}

	for _, attr := range doc.Attributes {
		switch a := attr.(type) {
		case *tg.DocumentAttributeAudio:
			return int(a.Duration)
		case *tg.DocumentAttributeVideo:
			return int(a.Duration)
		}
	}

	if len(doc.Attributes) > 0 {
		log.Printf("No supported duration attributes found: %T", media)
	} else {
		log.Print("No attributes found in the document.")
	}

	return 0
}

type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// GetMediaDuration returns duration in seconds (int).
func GetMediaDuration(input string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	args := []string{
		"-v", "error",
		"-print_format", "json",
		"-show_entries", "format=duration",
		input,
	}

	cmd := exec.CommandContext(ctx, "ffprobe", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		log.Printf("ffprobe timeout exceeded for %s", input)
		return 0
	}

	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		log.Printf("ffprobe failed: %s", msg)
		return 0
	}

	var out ffprobeOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		log.Printf("ffprobe failed: %s", err)
		return 0
	}

	if out.Format.Duration == "" {
		log.Print("ffprobe succeeded but duration not found")
		return 0
	}

	dur, err := strconv.ParseFloat(out.Format.Duration, 64)
	if err != nil {
		log.Printf("ffprobe failed: %s", err)
		return 0
	}

	return int(dur + 0.5)
}

// SecToMin converts a duration in seconds to a formatted string (MM:SS or HH:MM:SS).
func SecToMin(seconds int) string {
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
