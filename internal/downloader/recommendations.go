/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package downloader

import (
	"ashokshau/tgmusic/internal/utils"
	"context"
	"errors"
	"fmt"
	"strings"
)

func GetRecommendations(ctx context.Context, targetURL string, limit int) ([]utils.GetUrlTrack, error) {
	if limit <= 0 {
		limit = 15
	}

	urlToUse := strings.TrimSpace(targetURL)
	if urlToUse == "" {
		return nil, errors.New("no track URL or query provided for recommendations")
	}

	if len(urlToUse) == 11 && !strings.ContainsAny(urlToUse, " /?:") {
		urlToUse = "https://www.youtube.com/watch?v=" + urlToUse
	}
	seedID := extractVideoID(urlToUse)
	if !strings.HasPrefix(strings.ToLower(urlToUse), "http://") && !strings.HasPrefix(strings.ToLower(urlToUse), "https://") {
		matches, err := searchYouTube(urlToUse, 1, ctx)
		if err != nil {
			return nil, fmt.Errorf("find recommendation seed: %w", err)
		}
		if len(matches) == 0 {
			return nil, errors.New("no matching track found for recommendations")
		}
		urlToUse = matches[0].Url
		seedID = matches[0].Id
	}

	wrapper := NewDlWrapper(urlToUse)
	recTracks, err := wrapper.GetRecommendations(ctx, urlToUse)
	if err != nil {
		return nil, err
	}
	if recTracks == nil || len(recTracks.Results) == 0 {
		return nil, errors.New("no recommendation tracks found")
	}

	seen := make(map[string]bool)
	var uniqueTracks []utils.GetUrlTrack
	for _, track := range recTracks.Results {
		if track.Url == urlToUse || (seedID != "" && track.Id == seedID) {
			continue
		}
		key := track.Url
		if key == "" {
			key = track.Id
		}
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		uniqueTracks = append(uniqueTracks, track)
		if len(uniqueTracks) >= limit {
			break
		}
	}

	if len(uniqueTracks) == 0 {
		return nil, errors.New("no unique recommendation tracks found")
	}

	return uniqueTracks, nil
}
