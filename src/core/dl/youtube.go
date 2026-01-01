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
	"log"
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
	"youtube":   regexp.MustCompile(`^(?:https?://)?(?:www\.)?youtube\.com/watch\?v=([\w-]{11})(?:[&#?].*)?$`),
	"youtu_be":  regexp.MustCompile(`^(?:https?://)?(?:www\.)?youtu\.be/([\w-]{11})(?:[?#].*)?$`),
	"yt_shorts": regexp.MustCompile(`^(?:https?://)?(?:www\.)?youtube\.com/shorts/([\w-]{11})(?:[?#].*)?$`),
	//"yt_live":   regexp.MustCompile(`^(?:https?://)?(?:www\.)?youtube\.com/live/([\w-]{11})(?:[?#].*)?$`),
}

// NewYouTubeData initializes a YouTubeData instance with pre-compiled regex patterns and a cleaned query.
func NewYouTubeData(query string) *YouTubeData {
	return &YouTubeData{
		Query:    clearQuery(query),
		ApiUrl:   strings.TrimRight(config.Conf.ApiUrl, "/"),
		APIKey:   config.Conf.ApiKey,
		Patterns: youtubePatterns,
	}
}

// clearQuery removes extraneous URL parameters and fragments from a given query string.
func clearQuery(query string) string {
	query = strings.SplitN(query, "#", 2)[0]
	query = strings.SplitN(query, "&", 2)[0]
	return strings.TrimSpace(query)
}

// normalizeYouTubeURL converts various YouTube URL formats (e.g., youtu.be, shorts) into a standard watch URL.
func (y *YouTubeData) normalizeYouTubeURL(url string) string {
	var videoID string
	switch {
	case strings.Contains(url, "youtu.be/"):
		parts := strings.SplitN(strings.SplitN(url, "youtu.be/", 2)[1], "?", 2)
		videoID = strings.SplitN(parts[0], "#", 2)[0]
	case strings.Contains(url, "youtube.com/shorts/"):
		parts := strings.SplitN(strings.SplitN(url, "youtube.com/shorts/", 2)[1], "?", 2)
		videoID = strings.SplitN(parts[0], "#", 2)[0]
	default:
		return url
	}
	return "https://www.youtube.com/watch?v=" + videoID
}

// extractVideoID parses a YouTube URL and extracts the video ID.
func (y *YouTubeData) extractVideoID(url string) string {
	url = y.normalizeYouTubeURL(url)
	for _, pattern := range y.Patterns {
		if match := pattern.FindStringSubmatch(url); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

// IsValid checks if the query string matches any of the known YouTube URL patterns.
func (y *YouTubeData) IsValid() bool {
	if y.Query == "" {
		log.Println("The query or patterns are empty.")
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
func (y *YouTubeData) GetInfo(_ context.Context) (utils.PlatformTracks, error) {
	if !y.IsValid() {
		return utils.PlatformTracks{}, errors.New("the provided URL is invalid or the platform is not supported")
	}

	y.Query = y.normalizeYouTubeURL(y.Query)
	videoID := y.extractVideoID(y.Query)
	if videoID == "" {
		return utils.PlatformTracks{}, errors.New("unable to extract the video ID")
	}

	tracks, err := searchYouTube(videoID)
	if err != nil {
		return utils.PlatformTracks{}, err
	}

	for _, track := range tracks {
		if track.Id == videoID {
			return utils.PlatformTracks{Results: []utils.MusicTrack{track}}, nil
		}
	}

	return utils.PlatformTracks{}, errors.New("no video results were found")
}

// Search performs a search for a track on YouTube.
func (y *YouTubeData) Search(_ context.Context) (utils.PlatformTracks, error) {
	tracks, err := searchYouTube(y.Query)
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
		params = append(params, "--extract-audio", "--audio-format", "best")
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
		log.Printf("Could not generate a random number: %v", err)
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
		log.Println("Error creating download: " + err.Error())
		return "", err
	}

	return down.Process()
}
