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

type apiData struct {
	Query    string
	ApiUrl   string
	APIKey   string
	Patterns map[string]*regexp.Regexp
}

var apiPatterns = map[string]*regexp.Regexp{
	utils.Apple:      regexp.MustCompile(`(?i)^https?:\/\/music\.apple\.com\/[a-zA-Z-]+\/(?:song\/(?:[^\/]+\/)?\d+|album\/[^\/]+\/\d+(?:\?i=\d+)?|playlist\/[^\/]+\/pl\.[\w.-]+|artist\/[^\/]+\/\d+)(?:\?.*)?$`),
	utils.Spotify:    regexp.MustCompile(`(?i)^(https?://)?([a-z0-9-]+\.)*spotify\.com/(track|playlist|album|artist)/[a-zA-Z0-9]+(\?.*)?$`),
	utils.JioSaavn:   regexp.MustCompile(`(?i)https?:\/\/(?:www\.)?(?:jiosaavn|saavn)\.com\/(?:s\/)?(song|album|playlist|featured)(?:\/[^\/]+)*\/([A-Za-z0-9_,-]+)(?:\/)?(?:\?.*)?$`),
	utils.Deezer:     regexp.MustCompile(`(?i)https?:\/\/(?:www\.)?deezer\.com\/(?:[a-z]{2}\/)?(track|album|playlist)\/(\d+)`),
	utils.SoundCloud: regexp.MustCompile(`(?i)^(https?://)?(www\.)?soundcloud\.com/[a-zA-Z0-9_-]+/(sets/)?[a-zA-Z0-9._-]+(\?.*)?$`),
	utils.Gaana:      regexp.MustCompile(`(?i)https?:\/\/(?:www\.)?gaana\.com\/(song|album|playlist|artist)\/([A-Za-z0-9\-]+)`),
	utils.Tidal:      regexp.MustCompile(`(?i)https?:\/\/(?:www\.|listen\.)?tidal\.com\/(?:browse\/)?(track|album|playlist)\/([a-zA-Z0-9-]+)(?:[\/?].*)?`),
	utils.MXPlayer:   regexp.MustCompile(`(?i)https?:\/\/(?:www\.)?mxplayer\.in\/(?:show|movie)\/.*`),
	utils.Twitch:     regexp.MustCompile(`(?i)https?:\/\/(?:www\.|m\.)?twitch\.tv\/(?:videos|[\w._-]+\/video)\/\d+`),
	utils.TwitchClip: regexp.MustCompile(
		`(?i)https?:\/\/(?:www\.|m\.)?(?:` +
			`twitch\.tv\/clip\/[\w-]+|` +
			`clips\.twitch\.tv\/[\w-]+|` +
			`twitch\.tv\/[\w-]+\/clip\/[\w-]+` +
			`)`,
	),
	utils.Kick:     regexp.MustCompile(`(?i)https?:\/\/(?:www\.)?kick\.com\/[\w._-]+\/videos\/[a-fA-F0-9-]+`),
	utils.KickClip: regexp.MustCompile(`(?i)https?:\/\/(?:www\.)?kick\.com\/[\w._-]+\/clips\/[\w-]+`),
}

func newApiData(query string) *apiData {
	return &apiData{
		Query:    strings.TrimSpace(query),
		ApiUrl:   strings.TrimRight(config.ApiUrl, "/"),
		APIKey:   config.ApiKey,
		Patterns: apiPatterns,
	}
}

func (a *apiData) isValid() bool {
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

func (a *apiData) getInfo() (*utils.PlatformTracks, error) {
	if !a.isValid() {
		log.Warn("API get info rejected an invalid or unsupported URL")
		return nil, errors.New("the provided URL is invalid or the platform is not supported")
	}

	fullURL := fmt.Sprintf("%s/api/get_url?%s", a.ApiUrl, url.Values{"url": {a.Query}}.Encode())
	resp, err := sendRequest(http.MethodGet, fullURL, map[string]string{"X-API-Key": a.APIKey})
	if err != nil {
		log.Errorf("API get info request failed: %v", err)
		return nil, fmt.Errorf("Unable to load track information right now. Please try again.")
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Errorf("API get info returned HTTP %d", resp.StatusCode)
		return nil, fmt.Errorf("Unable to load track information right now. Please try again.")
	}

	var data utils.PlatformTracks
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Errorf("API get info response decode failed: %v", err)
		return nil, fmt.Errorf("Unable to read track information right now. Please try again.")
	}
	return &data, nil
}

func (a *apiData) search() (*utils.PlatformTracks, error) {
	if a.isValid() {
		return a.getInfo()
	}

	fullURL := fmt.Sprintf("%s/api/search?%s", a.ApiUrl, url.Values{
		"query": {a.Query},
		"limit": {"5"},
	}.Encode())

	resp, err := sendRequest(http.MethodGet, fullURL, map[string]string{"X-API-Key": a.APIKey})
	if err != nil {
		log.Errorf("API search request failed: %v", err)
		return nil, fmt.Errorf("Search is unavailable right now. Please try again.")
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Errorf("API search returned HTTP %d", resp.StatusCode)
		return nil, fmt.Errorf("Search is unavailable right now. Please try again.")
	}

	var data utils.PlatformTracks
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Errorf("API search response decode failed: %v", err)
		return nil, fmt.Errorf("Unable to read search results right now. Please try again.")
	}

	return &data, nil
}

func (a *apiData) getTrack() (*utils.TrackInfo, error) {
	fullURL := fmt.Sprintf("%s/api/track?%s", a.ApiUrl, url.Values{"url": {a.Query}}.Encode())
	resp, err := sendRequest(http.MethodGet, fullURL, map[string]string{"X-API-Key": a.APIKey})
	if err != nil {
		log.Errorf("API get track request failed: %v", err)
		return nil, fmt.Errorf("Unable to load this track right now. Please try again.")
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Errorf("API get track returned HTTP %d", resp.StatusCode)
		return nil, fmt.Errorf("Unable to load this track right now. Please try again.")
	}

	var data utils.TrackInfo
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Errorf("API get track response decode failed: %v", err)
		return nil, fmt.Errorf("Unable to read this track right now. Please try again.")
	}

	return &data, nil
}

func (a *apiData) downloadTrack(info *utils.TrackInfo, video bool) (string, error) {
	// if the track is from YouTube and video:true
	yt := newYouTubeData(a.Query)
	if info.Platform == utils.YouTube && video {
		return yt.downloadTrack(info, video)
	}

	filePath, err := processDownload(info)
	if err != nil {
		if info.Platform == utils.YouTube {
			log.Warnf("API track download failed; trying YouTube downloader: %v", err)
			return yt.downloadTrack(info, video)
		}
		log.Errorf("API track download failed: %v", err)
		return "", fmt.Errorf("Unable to download this track. Please try another.")
	}

	if strings.Contains(a.ApiUrl, filePath) {
		path, err := downloadFile(filePath, "", false)
		if err != nil {
			log.Errorf("API media file download failed: %v", err)
			return "", fmt.Errorf("Unable to download this track right now. Please try again.")
		}
		return path, nil
	}

	return filePath, nil
}

// getRecommendations retrieves recommended tracks for a given track URL from the API gateway.
func (a *apiData) getRecommendations(ctx context.Context, rawURL string) (*utils.PlatformTracks, error) {
	if a.ApiUrl == "" || a.APIKey == "" {
		log.Warn("API recommendations unavailable: URL or API key is not configured")
		return nil, errors.New("API URL or API key is not configured")
	}

	urlToUse := rawURL
	if urlToUse == "" {
		urlToUse = a.Query
	}
	if urlToUse == "" {
		return nil, errors.New("no URL provided for recommendations")
	}

	fullURL := fmt.Sprintf("%s/api/recommendations?%s", a.ApiUrl, url.Values{"url": {urlToUse}}.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		log.Errorf("API recommendations request creation failed: %v", err)
		return nil, fmt.Errorf("Unable to load recommendations right now. Please try again.")
	}
	req.Header.Set("X-API-Key", a.APIKey)
	resp, err := client.Do(req)
	if err != nil {
		requestErr := err
		if urlErr, ok := errors.AsType[*url.Error](err); ok {
			requestErr = urlErr.Err
		}
		log.Errorf("API recommendations request failed: %v", requestErr)
		return nil, fmt.Errorf("Unable to load recommendations right now. Please try again.")
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Errorf("API recommendations returned HTTP %d", resp.StatusCode)
		return nil, fmt.Errorf("Unable to load recommendations right now. Please try again.")
	}

	var data utils.PlatformTracks
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Errorf("API recommendations response decode failed: %v", err)
		return nil, fmt.Errorf("Unable to read recommendations right now. Please try again.")
	}
	return &data, nil
}
