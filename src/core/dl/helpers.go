/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package dl

import (
	"ashokshau/tgmusic/src/utils"
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	downloadTimeout        = 40 * time.Second
	defaultDownloadDirPerm = 0755
)

var (
	errMissingCDNURL = errors.New("missing cdn url")
)

// Download encapsulates the information and context required for a download operation.
type Download struct {
	Track utils.TrackInfo
	ctx   context.Context
}

// NewDownload creates and validates a new Download instance.
func NewDownload(ctx context.Context, track utils.TrackInfo) (*Download, error) {
	if track.CdnURL == "" {
		return nil, errors.New("the CDN URL is missing")
	}

	return &Download{Track: track, ctx: ctx}, nil
}

// Process initiates the download process based on the track's platform.
func (d *Download) Process() (string, error) {
	switch {
	case d.Track.CdnURL == "":
		return "", errMissingCDNURL

	case d.Track.Key != "" && strings.EqualFold(d.Track.Platform, "spotify"):
		return d.processSpotify()

	default:
		return d.processDirectDL()
	}
}

// processDirectDL manages direct downloads and includes improved error handling.
func (d *Download) processDirectDL() (string, error) {
	// No need to download (ntgcalls can play with url)
	return d.Track.CdnURL, nil
}

var (
	sanitizeRegex = regexp.MustCompile(`[<>:"/\\|?*]`)
	filenameRegex = regexp.MustCompile(`filename\*?=(?:UTF-8'')?([^;]+)`)
)

// sanitizeFilename removes invalid characters from a filename to ensure it is safe for the filesystem.
func sanitizeFilename(fileName string) string {
	fileName = strings.ReplaceAll(fileName, "/", "")
	fileName = strings.ReplaceAll(fileName, "\\", "")
	fileName = sanitizeRegex.ReplaceAllString(fileName, "")
	fileName = strings.TrimSpace(fileName)
	return fileName
}

// extractFilename parses the Content-Disposition header to extract the original filename.
func extractFilename(contentDisp string) string {
	if contentDisp == "" {
		return ""
	}
	matches := filenameRegex.FindStringSubmatch(contentDisp)
	if len(matches) > 1 {
		decoded, err := url.QueryUnescape(matches[1])
		if err == nil {
			return decoded
		}
	}
	return ""
}
