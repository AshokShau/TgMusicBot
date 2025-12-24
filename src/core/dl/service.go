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
)

// MusicService defines a standard interface for interacting with various music services.
// This allows for a unified approach to handling different platforms like YouTube, Spotify, etc.
type MusicService interface {
	// IsValid determines if the service can handle the given query.
	IsValid() bool
	// GetInfo retrieves metadata for a track or playlist.
	GetInfo(ctx context.Context) (utils.PlatformTracks, error)
	// Search queries the service for a track.
	Search(ctx context.Context) (utils.PlatformTracks, error)
	// GetTrack fetches detailed information for a single track.
	GetTrack(ctx context.Context) (utils.TrackInfo, error)
	// downloadTrack handles the download of a track.
	downloadTrack(ctx context.Context, trackInfo utils.TrackInfo, video bool) (string, error)
}

// DownloaderWrapper provides a unified interface for music service interactions,
type DownloaderWrapper struct {
	Query   string
	Service MusicService
}

// NewDownloaderWrapper selects the appropriate MusicService based on the query format or configuration defaults.
func NewDownloaderWrapper(query string) *DownloaderWrapper {
	yt := NewYouTubeData(query)
	api := NewApiData(query)
	direct := NewDirectLink(query)
	var chosen MusicService
	if yt.IsValid() {
		chosen = yt
	} else if api.IsValid() {
		chosen = api
	} else if direct.IsValid() {
		chosen = direct
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
func (d *DownloaderWrapper) GetInfo(ctx context.Context) (utils.PlatformTracks, error) {
	return d.Service.GetInfo(ctx)
}

// Search performs a search by delegating the call to the wrapped service.
func (d *DownloaderWrapper) Search(ctx context.Context) (utils.PlatformTracks, error) {
	return d.Service.Search(ctx)
}

// GetTrack retrieves detailed track information by delegating the call to the wrapped service.
func (d *DownloaderWrapper) GetTrack(ctx context.Context) (utils.TrackInfo, error) {
	return d.Service.GetTrack(ctx)
}

// DownloadTrack downloads a track by delegating the call to the wrapped service.
// It returns the file path of the downloaded track or an error if the download fails.
func (d *DownloaderWrapper) DownloadTrack(ctx context.Context, info utils.TrackInfo, video bool) (string, error) {
	return d.Service.downloadTrack(ctx, info, video)
}
