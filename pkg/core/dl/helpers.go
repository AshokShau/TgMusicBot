package dl

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"tgmusic/pkg/core/cache"
)

const (
	downloadTimeout        = 300
	defaultDownloadDirPerm = 0755
)

var (
	tgURLRegex             = regexp.MustCompile(`^https?://t\.me/`)
	errMissingCDNURL       = errors.New("missing cdn url")
	errUnsupportedPlatform = errors.New("unsupported platform")
)

// Download encapsulates the information and context required for a download operation.
type Download struct {
	Track cache.TrackInfo
	ctx   context.Context
}

// NewDownload creates and validates a new Download instance.
// It returns an error if the track's CDN URL is missing.
func NewDownload(ctx context.Context, track cache.TrackInfo) (*Download, error) {
	if track.CdnURL == "" {
		return nil, errors.New("the CDN URL is missing")
	}
	return &Download{Track: track, ctx: ctx}, nil
}

// Process initiates the download process based on the track's platform.
// It returns the file path of the downloaded track or an error if the download fails.
func (d *Download) Process() (string, error) {
	switch {
	case d.Track.CdnURL == "":
		return "", errMissingCDNURL
	case strings.EqualFold(d.Track.Platform, "spotify"):
		return d.processSpotify()
	default:
		return d.processDirectDL()
	}
}

// processDirectDL manages direct downloads and includes improved error handling.
// It returns the file path of the downloaded track or an error if the download fails.
func (d *Download) processDirectDL() (string, error) {
	track := d.Track
	if tgURLRegex.MatchString(track.CdnURL) {
		return track.CdnURL, nil
	}

	filePath, err := DownloadFile(d.ctx, track.CdnURL, "", false)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

// sanitizeFilename removes invalid characters from a filename to ensure it is safe for the filesystem.
func sanitizeFilename(fileName string) string {
	// Remove path separators.
	fileName = strings.ReplaceAll(fileName, "/", "")
	fileName = strings.ReplaceAll(fileName, "\\", "")
	// Remove other invalid characters.
	fileName = regexp.MustCompile(`[<>:"/\\|?*]`).ReplaceAllString(fileName, "")
	// Trim leading and trailing whitespace.
	fileName = strings.TrimSpace(fileName)
	return fileName
}

// extractFilename parses the Content-Disposition header to extract the original filename.
// It supports both "filename=" and "filename*=" formats.
func extractFilename(contentDisp string) string {
	if contentDisp == "" {
		return ""
	}
	// Match both "filename=" and "filename*=" to support a wider range of servers.
	re := regexp.MustCompile(`filename\*?=(?:UTF-8'')?([^;]+)`)
	matches := re.FindStringSubmatch(contentDisp)
	if len(matches) > 1 {
		// URL-decode the filename to handle encoded characters.
		decoded, err := url.QueryUnescape(matches[1])
		if err == nil {
			return decoded
		}
	}
	return ""
}
