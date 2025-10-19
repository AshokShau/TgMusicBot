package dl

import (
	"context"

	"github.com/AshokShau/TgMusicBot/pkg/config"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
)

// MusicService defines a standard interface for interacting with various music services.
// This allows for a unified approach to handling different platforms like YouTube, Spotify, etc.
type MusicService interface {
	// IsValid determines if the service can handle the given query.
	IsValid() bool
	// GetInfo retrieves metadata for a track or playlist.
	GetInfo(ctx context.Context) (cache.PlatformTracks, error)
	// Search queries the service for a track.
	Search(ctx context.Context) (cache.PlatformTracks, error)
	// GetTrack fetches detailed information for a single track.
	GetTrack(ctx context.Context) (cache.TrackInfo, error)
	// downloadTrack handles the download of a track.
	downloadTrack(ctx context.Context, trackInfo cache.TrackInfo, video bool) (string, error)
}

// DownloaderWrapper provides a unified interface for music service interactions,
// wrapping a specific MusicService implementation and delegating calls to it.
type DownloaderWrapper struct {
	Query   string
	Service MusicService
}

// NewDownloaderWrapper selects the appropriate MusicService based on the query format or configuration defaults.
// It returns a new DownloaderWrapper configured with the chosen service.
func NewDownloaderWrapper(query string) *DownloaderWrapper {
	yt := NewYouTubeData(query)
	api := NewApiData(query)
	var chosen MusicService
	if yt.IsValid() {
		chosen = yt
	} else if api.IsValid() {
		chosen = api
	} else {
		switch config.Conf.DefaultService {
		case "spotify":
			chosen = api
		default:
			chosen = yt
		}
	}

	return &DownloaderWrapper{
		Query:   query,
		Service: chosen,
	}
}

// IsValid checks if the underlying service can handle the query.
func (d *DownloaderWrapper) IsValid() bool {
	return d.Service != nil && d.Service.IsValid()
}

// GetInfo retrieves metadata by delegating the call to the wrapped service.
func (d *DownloaderWrapper) GetInfo(ctx context.Context) (cache.PlatformTracks, error) {
	return d.Service.GetInfo(ctx)
}

// Search performs a search by delegating the call to the wrapped service.
func (d *DownloaderWrapper) Search(ctx context.Context) (cache.PlatformTracks, error) {
	return d.Service.Search(ctx)
}

// GetTrack retrieves detailed track information by delegating the call to the wrapped service.
func (d *DownloaderWrapper) GetTrack(ctx context.Context) (cache.TrackInfo, error) {
	return d.Service.GetTrack(ctx)
}

// DownloadTrack downloads a track by delegating the call to the wrapped service.
// It returns the file path of the downloaded track or an error if the download fails.
func (d *DownloaderWrapper) DownloadTrack(ctx context.Context, info cache.TrackInfo, video bool) (string, error) {
	return d.Service.downloadTrack(ctx, info, video)
}
