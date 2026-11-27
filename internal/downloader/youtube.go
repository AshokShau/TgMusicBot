/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package downloader

import (
	"ashokshau/tgmusic/internal/config"
	"ashokshau/tgmusic/internal/utils"
	"time"

	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// youTubeData provides an interface for fetching track and playlist information from YouTube.
type youTubeData struct {
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

// newYouTubeData initializes a youTubeData instance with pre-compiled regex patterns and a cleaned query.
func newYouTubeData(query string) *youTubeData {
	return &youTubeData{
		Query:    strings.TrimSpace(query),
		ApiUrl:   strings.TrimRight(config.ApiUrl, "/"),
		APIKey:   config.ApiKey,
		Patterns: youtubePatterns,
	}
}

func (y *youTubeData) isValid() bool {
	if y.Query == "" {
		log.Warn("YouTube query is empty")
		return false
	}

	for _, pattern := range y.Patterns {
		if pattern.MatchString(y.Query) {
			return true
		}
	}
	return false
}

func (y *youTubeData) getInfo() (*utils.PlatformTracks, error) {
	if !y.isValid() {
		return nil, errors.New("the provided URL is invalid or the platform is not supported")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	y.Query = normalizeYouTubeURL(y.Query)
	videoID := extractVideoID(y.Query)
	playlistID := extractPlaylistID(y.Query)

	switch {
	case playlistID != "":
		if strings.HasPrefix(playlistID, "RD") && !strings.HasPrefix(playlistID, "RDCLAK") {
			return GetYouTubeMixPlaylist(ctx, playlistID)
		}
		return getYouTubePlaylist(ctx, playlistID)

	case videoID != "":
		var fallback *utils.GetUrlTrack
		if track, err := getYouTubeTrackFromOEmbed(ctx, videoID); err == nil {
			fallback = &track
		}
		queries := []string{videoID, y.Query}
		if fallback != nil {
			queries = append(queries, fallback.Title)
		}
		for _, query := range queries {
			tracks, err := searchYouTube(query, 10, ctx)
			if err != nil {
				continue
			}

			for _, track := range tracks {
				if track.Id == videoID {
					return &utils.PlatformTracks{Results: []utils.GetUrlTrack{track}}, nil
				}
			}
		}

		if fallback != nil {
			return &utils.PlatformTracks{Results: []utils.GetUrlTrack{*fallback}}, nil
		}

		log.Warnf("YouTube search returned no match for video %s", videoID)
		return getYouTubeVideo(ctx, videoID)
	}

	return nil, errors.New("no video or playlist results were found")
}

func (y *youTubeData) search() (*utils.PlatformTracks, error) {
	if y.isValid() {
		return y.getInfo()
	}

	tracks, err := searchYouTube(y.Query, 5)
	if err != nil {
		return nil, err
	}

	return &utils.PlatformTracks{Results: tracks}, nil
}

func (y *youTubeData) getTrack() (*utils.TrackInfo, error) {
	if y.Query == "" {
		return nil, errors.New("the query is empty")
	}

	if !y.isValid() {
		return nil, errors.New("the provided URL is invalid or the platform is not supported")
	}

	if y.ApiUrl != "" && y.APIKey != "" {
		if trackInfo, err := newApiData(y.Query).getTrack(); err == nil {
			return trackInfo, nil
		} else {
			log.Warnf("API track lookup failed; trying YouTube metadata: %v", err)
		}
	}

	getInfo, err := y.getInfo()
	if err != nil {
		return nil, err
	}
	if len(getInfo.Results) == 0 {
		return nil, errors.New("no video results were found")
	}

	track := getInfo.Results[0]
	trackInfo := &utils.TrackInfo{
		Id:       track.Id,
		URL:      track.Url,
		Platform: utils.YouTube,
	}

	return trackInfo, nil
}

// downloadTrack handles the download of a track from YouTube.
func (y *youTubeData) downloadTrack(info *utils.TrackInfo, video bool) (string, error) {
	if !video && info.CdnURL != "" {
		return info.CdnURL, nil
	}

	if !video && y.ApiUrl != "" && y.APIKey != "" {
		if filePath, err := y.downloadWithApi(info.Id, video); err == nil {
			return filePath, nil
		} else {
			log.Warnf("API download failed for video %s; trying yt-dlp: %v", info.Id, err)
		}
	}

	return y.downloadWithYtDlp(info.Id, video)
}

// buildYtdlpParams constructs the command-line parameters for yt-dlp to download media.
func (y *youTubeData) buildYtdlpParams(videoID string, video bool) ([]string, string) {
	outputTemplate := filepath.Join(config.DownloadsDir, "%(id)s.%(ext)s")
	var cookieFile string

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
		params = append(params, "-f", formatSelector, "--merge-output-format", "mkv")
	} else {
		params = append(params, "-f", "bestaudio[acodec=opus]/bestaudio[ext=m4a]/bestaudio")
	}

	cookieFile = y.getCookieFile()
	if cookieFile != "" {
		params = append(params, "--cookies", cookieFile)
	} else if config.Proxy != "" {
		params = append(params, "--proxy", config.Proxy)
	}

	videoURL := "https://www.youtube.com/watch?v=" + videoID
	params = append(params, videoURL, "--print", "after_move:filepath")

	return params, cookieFile
}

// downloadWithYtDlp downloads media from YouTube using the yt-dlp command-line tool.
func (y *youTubeData) downloadWithYtDlp(videoID string, video bool) (string, error) {
	if videoID == "" {
		return "", errors.New("videoID is empty")
	}

	ytdlpParams, cookieFile := y.buildYtdlpParams(videoID, video)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, ytdlpParams[0], ytdlpParams[1:]...)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			stderr := string(exitErr.Stderr)
			if cookieFile != "" && strings.Contains(stderr, "Sign in to confirm you're not a bot") {
				_ = os.Remove(cookieFile)
			}
			lower := strings.ToLower(stderr)
			if strings.Contains(lower, "private video") || strings.Contains(lower, "age-restricted") || strings.Contains(lower, "video unavailable") || strings.Contains(lower, "sign in to confirm you're not a bot") {
				log.Warnf("yt-dlp rejected video %s (exit %d): %s", videoID, exitErr.ExitCode(), stderr)
			} else {
				log.Errorf("yt-dlp download failed for video %s (exit %d): %s", videoID, exitErr.ExitCode(), stderr)
			}
			switch {
			case strings.Contains(lower, "sign in to confirm you're not a bot"), strings.Contains(lower, "bot detection"):
				return "", fmt.Errorf("YouTube restricted access for this track. Please try again later.")
			case strings.Contains(lower, "private video"):
				return "", fmt.Errorf("This video is private and cannot be played.")
			case strings.Contains(lower, "age-restricted"), strings.Contains(lower, "confirm your age"):
				return "", fmt.Errorf("This video is age-restricted and cannot be played.")
			case strings.Contains(lower, "video unavailable"), strings.Contains(lower, "this video has been removed"):
				return "", fmt.Errorf("This track is unavailable or has been removed.")
			default:
				return "", fmt.Errorf("YouTube could not download this track. Please try another.")
			}
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Warnf("yt-dlp timed out for video %s", videoID)
			return "", fmt.Errorf("YouTube download timed out. Please try again.")
		}

		log.Errorf("yt-dlp download failed for video %s: %v", videoID, err)
		return "", fmt.Errorf("YouTube could not download this track. Please try another.")
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
func (y *youTubeData) getCookieFile() string {
	cookiesPath := config.CookieFiles()
	if len(cookiesPath) == 0 {
		return ""
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(cookiesPath))))
	if err != nil {
		log.Warnf("select YouTube cookie file: %v", err)
		return cookiesPath[0]
	}

	return cookiesPath[n.Int64()]
}

// downloadWithApi downloads a track using the external API.
func (y *youTubeData) downloadWithApi(videoID string, _ bool) (string, error) {
	videoUrl := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	api := newApiData(videoUrl)
	track, err := api.getTrack()
	if err != nil {
		return "", err
	}

	return processDownload(track)
}

// getRecommendations retrieves native YouTube mix recommendations without external API.
func (y *youTubeData) getRecommendations(ctx context.Context, rawURL string) (*utils.PlatformTracks, error) {
	urlToUse := rawURL
	if urlToUse == "" {
		urlToUse = y.Query
	}

	videoID := extractVideoID(urlToUse)
	if videoID == "" && urlToUse != "" && !strings.HasPrefix(urlToUse, "http") {
		if len(urlToUse) == 11 {
			videoID = urlToUse
		}
	}

	if videoID == "" {
		return nil, errors.New("no valid YouTube video ID found for recommendations")
	}

	return GetYouTubeMixPlaylist(ctx, "RD"+videoID)
}
