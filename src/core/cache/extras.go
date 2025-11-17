/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// FFProbeFormat defines the structure for parsing the format information from ffprobe's JSON output.
type FFProbeFormat struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// GetFileDur extracts the duration of a media file from a Telegram message.
// It returns the duration in seconds or 0 if the media type is unsupported or has no duration.
func GetFileDur(m *tg.NewMessage) int {
	if !m.IsMedia() {
		return 0
	}

	switch media := m.Media().(type) {
	case *tg.MessageMediaDocument:
		return getDocumentDuration(media)
	case *tg.MessageMediaPhoto:
		return 0 // Photos do not have a duration.
	default:
		m.Client.Logger.Info("Unsupported media type: %T", media)
		return 0
	}
}

// getDocumentDuration extracts the duration from a document's attributes.
// It returns the duration in seconds or 0 if no duration attribute is found.
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
		log.Printf("No supported duration attributes found in (%T): %#v", media, doc.Attributes)
	} else {
		log.Print("No attributes found in the document.")
	}

	return 0
}

// GetFileDuration uses ffprobe to determine the duration of a media file.
// It takes a file path and returns the duration in seconds, or 0 if an error occurs.
func GetFileDuration(filePath string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to get audio duration with ffprobe: %v", err)
		return 0
	}

	var info FFProbeFormat
	if err := json.Unmarshal(output, &info); err != nil {
		log.Printf("Failed to parse ffprobe's JSON output: %v", err)
		return 0
	}

	var duration float64
	if info.Format.Duration != "" {
		if _, err := fmt.Sscanf(info.Format.Duration, "%f", &duration); err != nil {
			log.Printf("Could not parse duration format: %v", err)
			return 0
		}
	}

	return int(duration)
}
