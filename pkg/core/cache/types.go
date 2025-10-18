// Copyright (c) 2025 AshokShau
// Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
// Part of the TgMusicBot project. All rights reserved where applicable.

package cache

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
	Lyrics    string `json:"lyrics"`
	IsVideo   bool   `json:"is_video"`
	Platform  string `json:"platform"`
}

// TrackInfo holds detailed information about a specific track, including its CDN URL, cover art, and lyrics.
type TrackInfo struct {
	URL      string `json:"url"`
	CdnURL   string `json:"cdnurl"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	TC       string `json:"tc"`
	Cover    string `json:"cover"`
	Duration int    `json:"duration"`
	Lyrics   string `json:"lyrics"`
	Platform string `json:"platform"`
}

// MusicTrack represents a single music track returned from a search query.
// It contains essential details like the track's name, ID, and cover art URL.
type MusicTrack struct {
	URL      string `json:"url"`
	Name     string `json:"name"`
	ID       string `json:"id"`
	Cover    string `json:"cover"`
	Duration int    `json:"duration"`
	Platform string `json:"platform"`
}

// PlatformTracks is a collection of music tracks, typically returned from a search operation.
type PlatformTracks struct {
	Results []MusicTrack `json:"results"`
}

const (
	Telegram = "telegram"
	YouTube  = "youtube"
	Spotify  = "spotify"
	JioSaavn = "jiosaavn"
	Apple    = "apple_music"
)

const (
	Admins   = "admins"
	Everyone = "everyone"
	Auth     = "auth"
)
