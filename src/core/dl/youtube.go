/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package dl

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/utils"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// YouTubeData provides an interface for fetching track and playlist information from YouTube.
type YouTubeData struct {
	Query    string
	ApiUrl   string
	APIKey   string
	Patterns map[string]*regexp.Regexp
}

var youtubePatterns = map[string]*regexp.Regexp{
	"youtube":   regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtube\.com/.*`),
	"youtu_be":  regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtu\.be/.*`),
	"yt_music":  regexp.MustCompile(`(?i)^(?:https?://)?music\.youtube\.com/.*`),
	"yt_shorts": regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtube\.com/shorts/.*`),
}

// NewYouTubeData initializes a YouTubeData instance with pre-compiled regex patterns and a cleaned query.
func NewYouTubeData(query string) *YouTubeData {
	return &YouTubeData{
		Query:    strings.TrimSpace(query),
		ApiUrl:   strings.TrimRight(config.Conf.ApiUrl, "/"),
		APIKey:   config.Conf.ApiKey,
		Patterns: youtubePatterns,
	}
}


// IsValid checks if the query string matches any of the known YouTube URL patterns.
func (y *YouTubeData) IsValid() bool {
	if y.Query == "" {
		slog.Info("The query or patterns are empty.")
		return false
	}

	for _, pattern := range y.Patterns {
		if pattern.MatchString(y.Query) {
			return true
		}
	}
	return false
}

// GetInfo retrieves metadata for a track from YouTube.
// It returns a PlatformTracks object or an error if the information cannot be fetched.
func (y *YouTubeData) GetInfo(ctx context.Context) (utils.PlatformTracks, error) {
	if !y.IsValid() {
		return utils.PlatformTracks{}, errors.New("the provided URL is invalid or the platform is not supported")
	}

	y.Query = normalizeYouTubeURL(y.Query)
	videoID := extractVideoID(y.Query)
	playlistID := extractPlaylistID(y.Query)

	switch {
	case playlistID != "":
		if strings.HasPrefix(playlistID, "RD") {
			return GetYouTubeMixPlaylist(ctx, playlistID)
		}
		return GetYouTubePlaylist(ctx, playlistID)

	case videoID != "":
		for _, query := range []string{videoID, y.Query} {
			tracks, err := searchYouTube(query, 20)
			if err != nil {
				continue
			}

			for _, track := range tracks {
				if track.Id == videoID {
					return utils.PlatformTracks{Results: []utils.MusicTrack{track}}, nil
				}
			}
		}

		return GetYouTubeVideo(ctx, videoID)
	}

	return utils.PlatformTracks{}, errors.New("no video or playlist results were found")
}

// Search performs a search for a track on YouTube.
func (y *YouTubeData) Search(_ context.Context) (utils.PlatformTracks, error) {
	tracks, err := searchYouTube(y.Query, 20)
	if err != nil {
		return utils.PlatformTracks{}, err
	}
	if len(tracks) == 0 {
		return utils.PlatformTracks{}, errors.New("no video results were found")
	}
	return utils.PlatformTracks{Results: tracks}, nil
}

// GetTrack retrieves detailed information for a single track.
func (y *YouTubeData) GetTrack(ctx context.Context) (utils.TrackInfo, error) {
	if y.Query == "" {
		return utils.TrackInfo{}, errors.New("the query is empty")
	}
	if !y.IsValid() {
		return utils.TrackInfo{}, errors.New("the provided URL is invalid or the platform is not supported")
	}

	if y.ApiUrl != "" && y.APIKey != "" {
		if trackInfo, err := NewApiData(y.Query).GetTrack(ctx); err == nil {
			return trackInfo, nil
		}
	}

	getInfo, err := y.GetInfo(ctx)
	if err != nil {
		return utils.TrackInfo{}, err
	}
	if len(getInfo.Results) == 0 {
		return utils.TrackInfo{}, errors.New("no video results were found")
	}

	track := getInfo.Results[0]
	trackInfo := utils.TrackInfo{
		Id:       track.Id,
		URL:      track.Url,
		Platform: utils.YouTube,
	}

	return trackInfo, nil
}

// downloadTrack handles the download of a track from YouTube.
func (y *YouTubeData) downloadTrack(ctx context.Context, info utils.TrackInfo, video bool) (string, error) {
	if !video && y.ApiUrl != "" && y.APIKey != "" {
		if filePath, err := y.downloadWithApi(ctx, info.Id, video); err == nil {
			return filePath, nil
		}
	}

	filePath, err := y.downloadWithYtDlp(ctx, info.Id, video)
	return filePath, err
}

// buildYtdlpParams constructs the command-line parameters for yt-dlp to download media.
func (y *YouTubeData) buildYtdlpParams(videoID string, video bool) []string {
	outputTemplate := filepath.Join(config.Conf.DownloadsDir, "%(id)s.%(ext)s")

	params := []string{
		"yt-dlp",
		"--no-warnings",
		"--quiet",
		"--geo-bypass",
		"--retries", "2",
		"--continue",
		"--no-part",
		"--concurrent-fragments", "3",
		"--socket-timeout", "10",
		"--throttled-rate", "100K",
		"--retry-sleep", "1",
		"--no-write-thumbnail",
		"--no-write-info-json",
		"--no-embed-metadata",
		"--no-embed-chapters",
		"--no-embed-subs",
		"--extractor-args", "youtube:player_js_version=actual",
		"-o", outputTemplate,
	}

	if video {
		formatSelector := "bestvideo[height<=720]+bestaudio/best[height<=720]"
		params = append(params, "-f", formatSelector, "--merge-output-format", "mp4")
	} else {
		params = append(params, "-f", "bestaudio[ext=m4a]/bestaudio")
	}

	if cookieFile := y.getCookieFile(); cookieFile != "" {
		params = append(params, "--cookies", cookieFile)
	} else if config.Conf.Proxy != "" {
		params = append(params, "--proxy", config.Conf.Proxy)
	}

	videoURL := "https://www.youtube.com/watch?v=" + videoID
	params = append(params, videoURL, "--print", "after_move:filepath")

	return params
}

// downloadWithYtDlp downloads media from YouTube using the yt-dlp command-line tool.
func (y *YouTubeData) downloadWithYtDlp(ctx context.Context, videoID string, video bool) (string, error) {
	if videoID == "" {
		return "", errors.New("videoID is empty")
	}

	ytdlpParams := y.buildYtdlpParams(videoID, video)
	cmd := exec.CommandContext(ctx, ytdlpParams[0], ytdlpParams[1:]...)

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := string(exitErr.Stderr)
			return "", fmt.Errorf("yt-dlp failed with exit code %d: %s", exitErr.ExitCode(), stderr)
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("yt-dlp timed out for video ID: %s", videoID)
		}

		return "", fmt.Errorf("an unexpected error occurred while downloading %s: %w", videoID, err)
	}

	downloadedPathStr := strings.TrimSpace(string(output))
	if downloadedPathStr == "" {
		return "", fmt.Errorf("no output path was returned for %s", videoID)
	}

	if _, err := os.Stat(downloadedPathStr); os.IsNotExist(err) {
		return "", fmt.Errorf("the file was not found at the reported path: %s", downloadedPathStr)
	}

	return downloadedPathStr, nil
}

// getCookieFile retrieves the path to a cookie file from the configured list.
func (y *YouTubeData) getCookieFile() string {
	cookiesPath := config.Conf.CookiesPath
	if len(cookiesPath) == 0 {
		return ""
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(cookiesPath))))
	if err != nil {
		slog.Info("Could not generate a random number", "error", err)
		return cookiesPath[0]
	}

	return cookiesPath[n.Int64()]
}

// downloadWithApi downloads a track using the external API.
func (y *YouTubeData) downloadWithApi(ctx context.Context, videoID string, _ bool) (string, error) {
	videoUrl := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	api := NewApiData(videoUrl)
	track, err := api.GetTrack(ctx)
	if err != nil {
		return "", err
	}

	down, err := NewDownload(ctx, track)
	if err != nil {
		slog.Info("Error creating download: " + err.Error())
		return "", err
	}

	return down.Process()
}
