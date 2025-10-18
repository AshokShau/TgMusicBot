package dl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"tgmusic/pkg/config"
	"tgmusic/pkg/core/cache"

	"github.com/Laky-64/gologging"
)

// ApiData provides a unified interface for fetching track and playlist information from various music platforms via an API gateway.
type ApiData struct {
	Query    string
	ApiUrl   string
	APIKey   string
	Patterns map[string]*regexp.Regexp
}

// NewApiData creates and initializes a new ApiData instance with the provided query.
func NewApiData(query string) *ApiData {
	return &ApiData{
		Query:  strings.TrimSpace(query),
		ApiUrl: strings.TrimRight(config.Conf.ApiUrl, "/"),
		APIKey: config.Conf.ApiKey,
		Patterns: map[string]*regexp.Regexp{
			"apple_music": regexp.MustCompile(`(?i)^(https?://)?([a-z0-9-]+\.)*music\.apple\.com/([a-z]{2}/)?(album|playlist|song)/[a-zA-Z0-9\-._]+/(pl\.[a-zA-Z0-9]+|\d+)(\?.*)?$`),
			"spotify":     regexp.MustCompile(`(?i)^(https?://)?([a-z0-9-]+\.)*spotify\.com/(track|playlist|album|artist)/[a-zA-Z0-9]+(\?.*)?$`),
			"yt_playlist": regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?(?:youtube\.com|music\.youtube\.com)/(?:playlist|watch)\?.*\blist=([\w-]+)`),
			"yt_music":    regexp.MustCompile(`(?i)^(?:https?://)?music\.youtube\.com/(?:watch|playlist)\?.*v=([\w-]+)`),
			"jiosaavn":    regexp.MustCompile(`(?i)^(https?://)?(www\.)?jiosaavn\.com/(song|featured)/[\w-]+/[a-zA-Z0-9_-]+$`),
			"soundcloud":  regexp.MustCompile(`(?i)^(https?://)?([a-z0-9-]+\.)*soundcloud\.com/[a-zA-Z0-9_-]+(/(sets)?/[a-zA-Z0-9_-]+)?(\?.*)?$`),
		},
	}
}

// IsValid checks if the query is a valid URL for any of the supported platforms.
// It returns true if the URL matches a known pattern, and false otherwise.
func (a *ApiData) IsValid() bool {
	if a.Query == "" || a.ApiUrl == "" || a.APIKey == "" {
		gologging.WarnF("The query, API URL, or API key is missing.")
		return false
	}
	for name, pattern := range a.Patterns {
		if pattern.MatchString(a.Query) {
			gologging.DebugF("The platform has been matched: %s\n", name)
			return true
		}
	}
	return false
}

// GetInfo retrieves metadata for a track or playlist from the API.
// It returns a PlatformTracks object or an error if the request fails.
func (a *ApiData) GetInfo(ctx context.Context) (cache.PlatformTracks, error) {
	if !a.IsValid() {
		return cache.PlatformTracks{}, errors.New("the provided URL is invalid or the platform is not supported")
	}

	fullURL := fmt.Sprintf("%s/get_url?%s", a.ApiUrl, url.Values{"url": {a.Query}}.Encode())
	resp, err := sendRequest(ctx, http.MethodGet, fullURL, nil, map[string]string{"X-API-Key": a.APIKey})
	if err != nil {
		return cache.PlatformTracks{}, fmt.Errorf("the GetInfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cache.PlatformTracks{}, fmt.Errorf("unexpected status code while fetching info: %s", resp.Status)
	}

	var data cache.PlatformTracks
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return cache.PlatformTracks{}, fmt.Errorf("failed to decode the GetInfo response: %w", err)
	}
	return data, nil
}

// Search queries the API for a track. The context can be used for timeouts or cancellations.
// If the query is a valid URL, it fetches the information directly.
// It returns a PlatformTracks object or an error if the search fails.
func (a *ApiData) Search(ctx context.Context) (cache.PlatformTracks, error) {
	if a.IsValid() {
		return a.GetInfo(ctx)
	}

	fullURL := fmt.Sprintf("%s/search?%s", a.ApiUrl, url.Values{
		"query": {a.Query},
		"limit": {"5"},
	}.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return cache.PlatformTracks{}, fmt.Errorf("failed to create the search request: %w", err)
	}
	req.Header.Set("X-API-Key", a.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return cache.PlatformTracks{}, fmt.Errorf("the search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cache.PlatformTracks{}, fmt.Errorf("unexpected status code during search: %s", resp.Status)
	}

	var data cache.PlatformTracks
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return cache.PlatformTracks{}, fmt.Errorf("failed to decode the search response: %w", err)
	}
	return data, nil
}

// GetTrack retrieves detailed information for a single track from the API.
// It returns a TrackInfo object or an error if the request fails.
func (a *ApiData) GetTrack(ctx context.Context) (cache.TrackInfo, error) {
	fullURL := fmt.Sprintf("%s/track?%s", a.ApiUrl, url.Values{"url": {a.Query}}.Encode())
	resp, err := sendRequest(ctx, http.MethodGet, fullURL, nil, map[string]string{"X-API-Key": a.APIKey})
	if err != nil {
		return cache.TrackInfo{}, fmt.Errorf("the GetTrack request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cache.TrackInfo{}, fmt.Errorf("unexpected status code while fetching the track: %s", resp.Status)
	}

	var data cache.TrackInfo
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return cache.TrackInfo{}, fmt.Errorf("failed to decode the GetTrack response: %w", err)
	}
	return data, nil
}

// downloadTrack downloads a track using the API. If the track is a YouTube video and video format is requested,
// it delegates the download to the YouTube downloader.
// It returns the file path of the downloaded track or an error if the download fails.
func (a *ApiData) downloadTrack(ctx context.Context, info cache.TrackInfo, video bool) (string, error) {
	if info.Platform == "youtube" && video {
		yt := NewYouTubeData(a.Query)
		return yt.downloadTrack(ctx, info, video)
	}

	downloader, err := NewDownload(ctx, info)
	if err != nil {
		return "", fmt.Errorf("failed to initialize the download: %w", err)
	}

	filePath, err := downloader.Process()
	if err != nil {
		if info.Platform == "youtube" {
			yt := NewYouTubeData(a.Query)
			return yt.downloadTrack(ctx, info, video)
		}
		return "", fmt.Errorf("the download process failed: %w", err)
	}
	return filePath, nil
}
