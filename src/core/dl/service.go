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
)

// musicService defines a standard interface for interacting with various music services.
// This allows for a unified approach to handling different platforms like YouTube, Spotify, etc.
type musicService interface {
	// isValid determines if the service can handle the given query.
	isValid() bool
	// getInfo retrieves metadata for a track or playlist.
	getInfo() (utils.PlatformTracks, error)
	// search queries the service for a track.
	search() (utils.PlatformTracks, error)
	// getTrack fetches detailed information for a single track.
	getTrack() (utils.TrackInfo, error)
	// downloadTrack handles the download of a track.
	downloadTrack(trackInfo utils.TrackInfo, video bool) (string, error)
}

// DownloaderWrapper provides a unified interface for music service interactions.
type DownloaderWrapper struct {
	service musicService
}

// NewDownloaderWrapper selects the appropriate musicService based on the query format or configuration defaults.
func NewDownloaderWrapper(query string) *DownloaderWrapper {
	yt := newYouTubeData(query)
	api := newApiData(query)
	direct := newDirectLink(query)

	var chosen musicService
	if yt.isValid() {
		chosen = yt
	} else if api.isValid() {
		chosen = api
	} else if direct.isValid() {
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
		service: chosen,
	}
}

// IsValid checks if the underlying service can handle the query.
func (d *DownloaderWrapper) IsValid() bool {
	return d.service != nil && d.service.isValid()
}

// GetInfo retrieves metadata by delegating the call to the wrapped service.
func (d *DownloaderWrapper) GetInfo() (utils.PlatformTracks, error) {
	return d.service.getInfo()
}

// Search performs a search by delegating the call to the wrapped service.
func (d *DownloaderWrapper) Search() (utils.PlatformTracks, error) {
	return d.service.search()
}

// GetTrack retrieves detailed track information by delegating the call to the wrapped service.
func (d *DownloaderWrapper) GetTrack() (utils.TrackInfo, error) {
	return d.service.getTrack()
}

// DownloadTrack downloads a track by delegating the call to the wrapped service.
// It returns the file path of the downloaded track or an error if the download fails.
func (d *DownloaderWrapper) DownloadTrack(info utils.TrackInfo, video bool) (string, error) {
	return d.service.downloadTrack(info, video)
}
