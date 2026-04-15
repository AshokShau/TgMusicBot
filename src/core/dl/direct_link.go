/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package dl

import (
	"time"

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

type directLink struct {
	query string
}

func newDirectLink(query string) *directLink {
	return &directLink{query: query}
}

// ssValid checks if the query looks like a valid URL.
func (d *directLink) isValid() bool {
	return strings.HasPrefix(d.query, "http://") || strings.HasPrefix(d.query, "https://")
}

func (d *directLink) getInfo() (utils.PlatformTracks, error) {
	if !d.isValid() {
		return utils.PlatformTracks{}, errors.New("invalid url")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		d.query,
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
		parts := strings.Split(d.query, "/")
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
		Url:      d.query,
		Id:       d.query,
		Platform: utils.DirectLink,
	}

	return utils.PlatformTracks{Results: []utils.MusicTrack{track}}, nil
}

func (d *directLink) search() (utils.PlatformTracks, error) {
	return d.getInfo()
}

func (d *directLink) getTrack() (utils.TrackInfo, error) {
	info, err := d.getInfo()
	if err != nil {
		return utils.TrackInfo{}, err
	}

	if len(info.Results) == 0 {
		return utils.TrackInfo{}, errors.New("no track found")
	}

	track := info.Results[0]
	return utils.TrackInfo{
		Id:       track.Id,
		URL:      track.Url,
		CdnURL:   track.Url,
		Platform: track.Platform,
	}, nil
}

func (d *directLink) downloadTrack(_ utils.TrackInfo, _ bool) (string, error) {
	return d.query, nil
}
