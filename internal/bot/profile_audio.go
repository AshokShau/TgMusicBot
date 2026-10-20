package bot

import (
	"ashokshau/tgmusic/internal/cache"
	"ashokshau/tgmusic/internal/config"
	"ashokshau/tgmusic/internal/utils"
	"strconv"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
)

var (
	profileTracksCache = cache.NewCache[[]utils.GetUrlTrack](30 * time.Minute)
	profileTracksMu    sync.Mutex
	profileRetryAt     time.Time
)

func getProfileTracks(c *td.Client, userID int64) []utils.GetUrlTrack {
	profileTracksMu.Lock()
	defer profileTracksMu.Unlock()

	key := strconv.FormatInt(userID, 10)
	if tracks, ok := profileTracksCache.Get(key); ok {
		return tracks
	}
	if time.Now().Before(profileRetryAt) {
		return nil
	}

	audios, err := c.GetUserProfileAudios(10, userID, nil)

	ttl := 5 * time.Minute
	var tracks []utils.GetUrlTrack
	if err == nil && audios != nil {
		seen := make(map[string]bool)
		for _, audio := range audios.Audios {
			file := audio.Audio
			if file == nil || file.Remote == nil || file.Remote.Id == "" ||
				file.Size > config.MaxFileSize || audio.Duration > config.SongDurationLimit || seen[file.Remote.Id] {
				continue
			}

			seen[file.Remote.Id] = true
			title := audio.Title
			if title == "" {
				title = audio.FileName
			}

			if title == "" {
				title = "Profile audio"
			}

			tracks = append(tracks, utils.GetUrlTrack{
				Id: file.Remote.Id, Title: title, Duration: audio.Duration,
				Channel: audio.Performer, Platform: utils.Telegram,
			})

			if len(tracks) == 10 {
				break
			}
		}

		if len(tracks) > 0 {
			ttl = 30 * time.Minute
		}
	} else if wait := getFloodWait(err); wait > 0 {
		profileRetryAt = time.Now().Add(time.Duration(wait) * time.Second)
		if backoff := time.Until(profileRetryAt); backoff > ttl {
			ttl = backoff
		}
	}

	profileTracksCache.SetWithTTL(key, tracks, ttl)
	return tracks
}
