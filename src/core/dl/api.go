/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package dl

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// ApiData provides a unified interface for fetching track and playlist information from various music platforms via an API gateway.
type ApiData struct {
	Query    string
	ApiUrl   string
	APIKey   string
	Patterns map[string]*regexp.Regexp
}

var apiPatterns = map[string]*regexp.Regexp{
	utils.Apple:      regexp.MustCompile(`(?i)^https?:\/\/music\.apple\.com\/[a-zA-Z-]+\/(?:song\/(?:[^\/]+\/)?\d+|album\/[^\/]+\/\d+(?:\?i=\d+)?|playlist\/[^\/]+\/pl\.[\w.-]+|artist\/[^\/]+\/\d+)(?:\?.*)?$`),
	utils.Spotify:    regexp.MustCompile(`(?i)^(https?://)?([a-z0-9-]+\.)*spotify\.com/(track|playlist|album|artist)/[a-zA-Z0-9]+(\?.*)?$`),
	"yt_live":        regexp.MustCompile(`^(?:https?://)?(?:www\.)?youtube\.com/live/([\w-]{11})(?:[?#].*)?$`),
	"yt_playlist":    regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?(?:youtube\.com|music\.youtube\.com)/(?:playlist|watch)\?.*\blist=([\w-]+)`),
	"yt_music":       regexp.MustCompile(`(?i)^(?:https?://)?music\.youtube\.com/(?:watch|playlist)\?.*v=([\w-]+)`),
	utils.JioSaavn:   regexp.MustCompile(`(?i)https?:\/\/(?:www\.)?jiosaavn\.com\/(song|album|playlist|featured)\/[^\/]+\/([A-Za-z0-9_]+)`),
	utils.Deezer:     regexp.MustCompile(`(?i)https?:\/\/(?:www\.)?deezer\.com\/(?:[a-z]{2}\/)?(track|album|playlist)\/(\d+)`),
	utils.SoundCloud: regexp.MustCompile(`(?i)^(https?://)?(www\.)?soundcloud\.com/[a-zA-Z0-9_-]+/(sets/)?[a-zA-Z0-9._-]+(\?.*)?$`),
}

// NewApiData creates and initializes a new ApiData instance with the provided query.
func NewApiData(query string) *ApiData {
	return &ApiData{
		Query:    strings.TrimSpace(query),
		ApiUrl:   strings.TrimRight(config.Conf.ApiUrl, "/"),
		APIKey:   config.Conf.ApiKey,
		Patterns: apiPatterns,
	}
}

// IsValid checks if the query is a valid URL for any of the supported platforms.
func (a *ApiData) IsValid() bool {
	if a.Query == "" || a.ApiUrl == "" || a.APIKey == "" {
		return false
	}

	for _, pattern := range a.Patterns {
		if pattern.MatchString(a.Query) {
			return true
		}
	}
	return false
}

// GetInfo retrieves metadata for a track or playlist from the API.
func (a *ApiData) GetInfo(ctx context.Context) (utils.PlatformTracks, error) {
	if !a.IsValid() {
		return utils.PlatformTracks{}, errors.New("the provided URL is invalid or the platform is not supported")
	}

	fullURL := fmt.Sprintf("%s/api/get_url?%s", a.ApiUrl, url.Values{"url": {a.Query}}.Encode())
	resp, err := sendRequest(ctx, http.MethodGet, fullURL, nil, map[string]string{"X-API-Key": a.APIKey})
	if err != nil {
		return utils.PlatformTracks{}, fmt.Errorf("the GetInfo request failed: %w", err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return utils.PlatformTracks{}, fmt.Errorf("unexpected status code while fetching info: %s", resp.Status)
	}

	var data utils.PlatformTracks
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return utils.PlatformTracks{}, fmt.Errorf("failed to decode the GetInfo response: %w", err)
	}
	return data, nil
}

// Search queries the API for a track. The context can be used for timeouts or cancellations.
func (a *ApiData) Search(ctx context.Context) (utils.PlatformTracks, error) {
	if a.IsValid() {
		return a.GetInfo(ctx)
	}

	fullURL := fmt.Sprintf("%s/api/search?%s", a.ApiUrl, url.Values{
		"query": {a.Query},
		"limit": {"5"},
	}.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return utils.PlatformTracks{}, fmt.Errorf("failed to create the search request: %w", err)
	}
	req.Header.Set("X-API-Key", a.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return utils.PlatformTracks{}, fmt.Errorf("the search request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return utils.PlatformTracks{}, fmt.Errorf("unexpected status code during search: %s", resp.Status)
	}

	var data utils.PlatformTracks
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return utils.PlatformTracks{}, fmt.Errorf("failed to decode the search response: %w", err)
	}
	return data, nil
}

// GetTrack retrieves detailed information for a single track from the API.
func (a *ApiData) GetTrack(ctx context.Context) (utils.TrackInfo, error) {
	fullURL := fmt.Sprintf("%s/api/track?%s", a.ApiUrl, url.Values{"url": {a.Query}}.Encode())
	resp, err := sendRequest(ctx, http.MethodGet, fullURL, nil, map[string]string{"X-API-Key": a.APIKey})
	if err != nil {
		return utils.TrackInfo{}, fmt.Errorf("the GetTrack request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return utils.TrackInfo{}, fmt.Errorf("unexpected status code while fetching the track: %s", resp.Status)
	}

	var data utils.TrackInfo
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return utils.TrackInfo{}, fmt.Errorf("failed to decode the GetTrack response: %w", err)
	}
	return data, nil
}

// downloadTrack downloads a track using the API. If the track is a YouTube video and video format is requested,
func (a *ApiData) downloadTrack(ctx context.Context, info utils.TrackInfo, video bool) (string, error) {
	yt := NewYouTubeData(a.Query)
	if info.Platform == utils.YouTube && video {
		return yt.downloadTrack(ctx, info, video)
	}

	downloader, err := NewDownload(ctx, info)
	if err != nil {
		return "", fmt.Errorf("failed to initialize the download: %w", err)
	}

	filePath, err := downloader.Process()
	if err != nil {
		if info.Platform == utils.YouTube {
			return yt.downloadTrack(ctx, info, video)
		}

		return "", fmt.Errorf("the download process failed: %w", err)
	}
	return filePath, nil
}
