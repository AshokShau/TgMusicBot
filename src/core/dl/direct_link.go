/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package dl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"ashokshau/tgmusic/src/core/cache"
)

type DirectLink struct {
	Query string
}

func NewDirectLink(query string) *DirectLink {
	return &DirectLink{Query: query}
}

// IsValid checks if the query looks like a valid URL.
func (d *DirectLink) IsValid() bool {
	return strings.HasPrefix(d.Query, "http://") || strings.HasPrefix(d.Query, "https://")
}

func (d *DirectLink) GetInfo(ctx context.Context) (cache.PlatformTracks, error) {
	if !d.IsValid() {
		return cache.PlatformTracks{}, errors.New("invalid url")
	}

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		d.Query,
	)

	output, err := cmd.Output()
	if err != nil {
		return cache.PlatformTracks{}, fmt.Errorf("invalid or unplayable link: %w", err)
	}

	var info cache.FFProbeFormat
	if err := json.Unmarshal(output, &info); err != nil {
		return cache.PlatformTracks{}, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	duration := 0
	if info.Format.Duration != "" {
		if d, err := strconv.ParseFloat(info.Format.Duration, 64); err == nil {
			duration = int(d)
		}
	}

	title := info.Format.Tags.Title
	if title == "" {
		parts := strings.Split(d.Query, "/")
		if len(parts) > 0 {
			title = parts[len(parts)-1]
			title = strings.SplitN(title, "?", 2)[0]
			title = strings.SplitN(title, "#", 2)[0]
			title, _ = url.QueryUnescape(title)
		}
		if title == "" {
			title = "Direct Link"
		}
	}

	const maxTitleLength = 30
	if len(title) > maxTitleLength {
		title = title[:maxTitleLength-3] + "..."
	}

	track := cache.MusicTrack{
		Name:     title,
		Duration: duration,
		URL:      d.Query,
		ID:       d.Query,
		Platform: cache.DirectLink,
	}

	return cache.PlatformTracks{Results: []cache.MusicTrack{track}}, nil
}

func (d *DirectLink) Search(ctx context.Context) (cache.PlatformTracks, error) {
	return d.GetInfo(ctx)
}

func (d *DirectLink) GetTrack(ctx context.Context) (cache.TrackInfo, error) {
	info, err := d.GetInfo(ctx)
	if err != nil {
		return cache.TrackInfo{}, err
	}
	if len(info.Results) == 0 {
		return cache.TrackInfo{}, errors.New("no track found")
	}

	t := info.Results[0]
	return cache.TrackInfo{
		URL:      d.Query,
		Name:     t.Name,
		Duration: t.Duration,
		Platform: cache.DirectLink,
		CdnURL:   d.Query,
		TC:       d.Query,
	}, nil
}

func (d *DirectLink) downloadTrack(_ context.Context, _ cache.TrackInfo, _ bool) (string, error) {
	return d.Query, nil
}
