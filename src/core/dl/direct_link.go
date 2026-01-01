/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package dl

import (
	"ashokshau/tgmusic/src/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
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

func (d *DirectLink) GetInfo(ctx context.Context) (utils.PlatformTracks, error) {
	if !d.IsValid() {
		return utils.PlatformTracks{}, errors.New("invalid url")
	}

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		d.Query,
	)

	output, err := cmd.Output()
	if err != nil {
		return utils.PlatformTracks{}, fmt.Errorf("invalid or unplayable link: %w", err)
	}

	var info utils.FFProbeFormat
	if err = json.Unmarshal(output, &info); err != nil {
		return utils.PlatformTracks{}, fmt.Errorf("failed to parse ffprobe output: %w", err)
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

	track := utils.MusicTrack{
		Title:    title,
		Duration: duration,
		Url:      d.Query,
		Id:       d.Query,
		Platform: utils.DirectLink,
	}

	return utils.PlatformTracks{Results: []utils.MusicTrack{track}}, nil
}

func (d *DirectLink) Search(ctx context.Context) (utils.PlatformTracks, error) {
	return d.GetInfo(ctx)
}

func (d *DirectLink) GetTrack(ctx context.Context) (utils.TrackInfo, error) {
	info, err := d.GetInfo(ctx)
	if err != nil {
		return utils.TrackInfo{}, err
	}

	if len(info.Results) == 0 {
		return utils.TrackInfo{}, errors.New("no track found")
	}

	return utils.TrackInfo{
		URL:      d.Query,
		Platform: utils.DirectLink,
		CdnURL:   d.Query,
	}, nil
}

func (d *DirectLink) downloadTrack(_ context.Context, _ utils.TrackInfo, _ bool) (string, error) {
	return d.Query, nil
}
