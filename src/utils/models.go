/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package utils

// CachedTrack defines the structure for a track that is stored in the queue.
// It includes metadata such as the track's URL, name, duration, and the user who requested it.
type CachedTrack struct {
	URL       string `json:"url"`
	Name      string `json:"name"`
	Loop      int    `json:"loop"`
	User      string `json:"user"`
	FilePath  string `json:"file_path"`
	Thumbnail string `json:"thumbnail"`
	TrackID   string `json:"track_id"`
	Duration  int    `json:"duration"`
	Channel   string `json:"channel"`
	Views     string `json:"views"`
	IsVideo   bool   `json:"is_video"`
	Platform  string `json:"platform"`
}

// TrackInfo holds detailed information about a specific track, including its CDN URL, cover art, and lyrics.
type TrackInfo struct {
	Id       string `json:"id"`
	URL      string `json:"url"`
	CdnURL   string `json:"cdnurl"`
	Key      string `json:"key"`
	Platform string `json:"platform"`
}

// MusicTrack represents a single music track returned from a search query.
// It contains essential details like the track's name, ID, and cover art URL.
type MusicTrack struct {
	Title     string `json:"title"`
	Id        string `json:"id"`
	Url       string `json:"url"`
	Thumbnail string `json:"thumbnail"`
	Duration  int    `json:"duration"`
	Channel   string `json:"channel"`
	Views     string `json:"views"`
	Platform  string `json:"platform"`
}

// PlatformTracks is a collection of music tracks, typically returned from a search operation.
type PlatformTracks struct {
	Results []MusicTrack `json:"results"`
}

const (
	Telegram   = "telegram"
	YouTube    = "youtube"
	Spotify    = "spotify"
	JioSaavn   = "jiosaavn"
	Apple      = "apple_music"
	SoundCloud = "soundcloud"
	Deezer     = "Deezer"
	Gaana      = "Gaana"
	DirectLink = "direct_link"
)

const (
	Admins   = "admins"
	Everyone = "everyone"
	Auth     = "auth"
)

// FFProbeFormat defines the structure for parsing the format information from ffprobe's JSON output.
type FFProbeFormat struct {
	Format struct {
		Duration string `json:"duration"`
		Tags     struct {
			Title string `json:"title"`
		} `json:"tags,omitempty"`
	} `json:"format"`
}
