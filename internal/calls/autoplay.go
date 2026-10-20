package calls

import (
	"ashokshau/tgmusic/internal/cache"
	"ashokshau/tgmusic/internal/config"
	"ashokshau/tgmusic/internal/downloader"
	"ashokshau/tgmusic/internal/utils"
	"context"
	"math/rand"
	"slices"
	"time"

	td "github.com/AshokShau/gotdbot"
)

func (c *TelegramCalls) handleAutoplay(bot *td.Client, chatID int64, lastTrack string, quiet bool) error {
	history := cache.ChatCache.GetAutoplayHistory(chatID)
	if len(history) >= int(config.AutoPlayLimit) {
		cache.ChatCache.ClearAutoplayHistory(chatID)
		return c.handleNoSong(bot, chatID, quiet)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tracks, err := downloader.GetRecommendations(ctx, lastTrack, int(config.AutoPlayLimit))
	if err != nil {
		_, _ = bot.SendMessage(chatID, "Unable to load recommended tracks. Try /play to add another song.", nil)
		return c.handleNoSong(bot, chatID, true)
	}
	if len(tracks) == 0 {
		return c.handleNoSong(bot, chatID, quiet)
	}

	var candidates []utils.GetUrlTrack
	for _, track := range tracks {
		if track.Id == lastTrack || track.Url == lastTrack || slices.Contains(history, track.Id) || slices.Contains(history, track.Url) ||
			(config.SongDurationLimit > 0 && track.Duration > config.SongDurationLimit) {
			continue
		}

		candidates = append(candidates, track)
	}

	if len(candidates) == 0 {
		return c.handleNoSong(bot, chatID, quiet)
	}

	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	nextTrack := candidates[0]
	trackKey := nextTrack.Url
	if trackKey == "" {
		trackKey = nextTrack.Id
	}
	cache.ChatCache.AddAutoplayHistory(chatID, lastTrack, trackKey)
	platform := nextTrack.Platform
	if platform == "" {
		platform = utils.YouTube
	}

	saveCache := &utils.PlayerCache{
		URL:       nextTrack.Url,
		Name:      nextTrack.Title,
		User:      utils.AutoPlay,
		Thumbnail: nextTrack.Thumbnail,
		TrackID:   nextTrack.Id,
		Duration:  nextTrack.Duration,
		Channel:   nextTrack.Channel,
		Views:     nextTrack.Views,
		Platform:  platform,
		Silent:    quiet,
	}

	cache.ChatCache.AddSong(chatID, saveCache)
	return c.playTrack(bot, chatID, saveCache)
}
