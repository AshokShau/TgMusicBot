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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	labelDurationRe = regexp.MustCompile(`(\d+)\s*(hour|minute|second)`)
	videoIDRe1      = regexp.MustCompile(`(?i)(?:youtube\.com/(?:watch\?v=|embed/|shorts/|live/)|youtu\.be/)([A-Za-z0-9_-]{11})`)
	videoIDRe2      = regexp.MustCompile(`(?:v=|\/)([0-9A-Za-z_-]{11})`)
	playlistIDRe1   = regexp.MustCompile(`(?i)(?:youtube\.com|music\.youtube\.com).*(?:\?|&)list=([A-Za-z0-9_-]+)`)
	playlistIDRe2   = regexp.MustCompile(`list=([0-9A-Za-z_-]+)`)
)

func searchYouTube(query string, limit int) ([]utils.MusicTrack, error) {
	endpoint := "https://www.youtube.com/youtubei/v1/search?key=AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"

	payload := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB",
				"clientVersion": "2.20250101.01.00",
				"hl":            "en",
				"gl":            "IN",
			},
		},
		"query": query,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf(
			"youtube search failed: status=%d %s body=%q",
			resp.StatusCode,
			resp.Status,
			string(raw),
		)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err = json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	root := dig(
		data,
		"contents",
		"twoColumnSearchResultsRenderer",
		"primaryContents",
		"sectionListRenderer",
		"contents",
	)

	var tracks []utils.MusicTrack
	parseResults(root, &tracks, limit)

	return tracks, nil
}

func parseResults(node interface{}, tracks *[]utils.MusicTrack, limit int) {
	if len(*tracks) >= limit {
		return
	}

	switch v := node.(type) {

	case []interface{}:
		for _, i := range v {
			parseResults(i, tracks, limit)
			if len(*tracks) >= limit {
				return
			}
		}

	case map[string]interface{}:
		if vr, ok := dig(v, "videoRenderer").(map[string]interface{}); ok {
			if badges, ok := vr["badges"].([]interface{}); ok {
				for _, badge := range badges {
					if meta, ok := dig(badge, "metadataBadgeRenderer").(map[string]interface{}); ok {
						if safeString(meta["style"]) == "BADGE_STYLE_TYPE_LIVE_NOW" {
							return
						}
					}
				}
			}

			id := safeString(vr["videoId"])
			title := safeString(dig(vr, "title", "runs", 0, "text"))
			durationText := safeString(dig(vr, "lengthText", "simpleText"))
			if id == "" || title == "" || durationText == "" {
				return
			}

			*tracks = append(*tracks, utils.MusicTrack{
				Id:        id,
				Url:       "https://www.youtube.com/watch?v=" + id,
				Title:     title,
				Thumbnail: safeString(dig(vr, "thumbnail", "thumbnails", 0, "url")),
				Duration:  parseDuration(durationText),
				Views:     safeString(dig(vr, "viewCountText", "simpleText")),
				Channel:   safeString(dig(vr, "ownerText", "runs", 0, "text")),
				Platform:  utils.YouTube,
			})
		}

		for _, c := range v {
			parseResults(c, tracks, limit)
		}
	}
}

func dig(v interface{}, path ...interface{}) interface{} {
	cur := v
	for _, p := range path {
		switch k := p.(type) {
		case string:
			m, ok := cur.(map[string]interface{})
			if !ok {
				return nil
			}
			cur = m[k]

		case int:
			a, ok := cur.([]interface{})
			if !ok || k < 0 || k >= len(a) {
				return nil
			}
			cur = a[k]
		}
	}
	return cur
}

func safeString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func normalizeYouTubeURL(url string) string {
	var videoID string
	switch {
	case strings.Contains(url, "youtu.be/"):
		parts := strings.SplitN(strings.SplitN(url, "youtu.be/", 2)[1], "?", 2)
		videoID = strings.SplitN(parts[0], "#", 2)[0]
	case strings.Contains(url, "youtube.com/shorts/"):
		parts := strings.SplitN(strings.SplitN(url, "youtube.com/shorts/", 2)[1], "?", 2)
		videoID = strings.SplitN(parts[0], "#", 2)[0]
	default:
		return url
	}
	return "https://www.youtube.com/watch?v=" + videoID
}

func parseDuration(s string) int {
	parts := strings.Split(s, ":")
	total := 0
	mul := 1
	for i := len(parts) - 1; i >= 0; i-- {
		total += atoi(parts[i]) * mul
		mul *= 60
	}
	return total
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	return n
}

func digStr(src any, path ...any) string {
	cur := src
	for _, p := range path {
		switch k := p.(type) {
		case string:
			m, ok := cur.(map[string]any)
			if !ok {
				return ""
			}
			cur = m[k]
		case int:
			a, ok := cur.([]any)
			if !ok || k >= len(a) {
				return ""
			}
			cur = a[k]
		}
	}
	s, _ := cur.(string)
	return s
}

func digArray(src any, path ...any) []map[string]any {
	cur := src

	for _, p := range path {
		switch k := p.(type) {
		case string:
			m, ok := cur.(map[string]any)
			if !ok {
				return nil
			}
			cur = m[k]
		case int:
			a, ok := cur.([]any)
			if !ok || k >= len(a) {
				return nil
			}
			cur = a[k]
		}
	}

	arr, ok := cur.([]any)
	if !ok {
		return nil
	}

	var out []map[string]any
	for _, v := range arr {
		if m, ok := v.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func pickYTThumb(v map[string]any) string {
	thumbs := digArray(v, "thumbnail", "thumbnails")
	if len(thumbs) == 0 {
		return ""
	}
	if t, ok := thumbs[len(thumbs)-1]["url"].(string); ok {
		return t
	}
	return ""
}

func parseTimeToSeconds(s string) int {
	parts := strings.Split(s, ":")
	total := 0
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return total
}

func parseLabelDuration(s string) int {
	matches := labelDurationRe.FindAllStringSubmatch(s, -1)
	total := 0
	for _, m := range matches {
		n, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "hour":
			total += n * 3600
		case "minute":
			total += n * 60
		case "second":
			total += n
		}
	}
	return total
}

func parseYTDuration(v map[string]any) int {
	txt := digStr(v, "lengthText", "simpleText")
	if txt != "" {
		return parseTimeToSeconds(txt)
	}
	label := digStr(v, "lengthText", "accessibility", "accessibilityData", "label")
	if label != "" {
		return parseLabelDuration(label)
	}
	return 0
}

func mapYTVideo(v map[string]any) utils.MusicTrack {
	id := digStr(v, "videoId")
	return utils.MusicTrack{
		Id:        id,
		Title:     digStr(v, "title", "runs", 0, "text"),
		Url:       "https://www.youtube.com/watch?v=" + id,
		Thumbnail: pickYTThumb(v),
		Channel:   digStr(v, "shortBylineText", "runs", 0, "text"),
		Duration:  parseYTDuration(v),
		Views:     digStr(v, "viewCountText", "simpleText"),
		Platform:  utils.YouTube,
	}
}

func mapMixVideo(v map[string]any) utils.MusicTrack {
	id := digStr(v, "videoId")
	return utils.MusicTrack{
		Id:        id,
		Title:     digStr(v, "title", "simpleText"),
		Url:       "https://www.youtube.com/watch?v=" + id,
		Thumbnail: pickYTThumb(v),
		Channel:   digStr(v, "shortBylineText", "runs", 0, "text"),
		Duration:  parseYTDuration(v),
		Platform:  utils.YouTube,
	}
}

func extractMixPlaylistVideos(src map[string]any) []map[string]any {
	var out []map[string]any
	contents := digArray(src, "contents", "twoColumnWatchNextResults", "playlist", "playlist", "contents")
	for _, c := range contents {
		if v, ok := c["playlistPanelVideoRenderer"].(map[string]any); ok {
			out = append(out, v)
		}
	}
	return out
}

func extractPlaylistVideos(src map[string]any) []map[string]any {
	var out []map[string]any
	contents := digArray(
		src,
		"contents",
		"twoColumnBrowseResultsRenderer",
		"tabs",
		0,
		"tabRenderer",
		"content",
		"sectionListRenderer",
		"contents",
		0,
		"itemSectionRenderer",
		"contents",
		0,
		"playlistVideoListRenderer",
		"contents",
	)
	for _, c := range contents {
		if v, ok := c["playlistVideoRenderer"].(map[string]any); ok {
			out = append(out, v)
		}
	}
	return out
}

func pickYTPlayerThumb(src map[string]any) string {
	thumbs := digArray(src, "videoDetails", "thumbnail", "thumbnails")
	if len(thumbs) == 0 {
		return ""
	}
	if t, ok := thumbs[len(thumbs)-1]["url"].(string); ok {
		return t
	}
	return ""
}

func mapPlayerToTrack(src map[string]any) utils.MusicTrack {
	videoID := digStr(src, "videoDetails", "videoId")
	return utils.MusicTrack{
		Id:        videoID,
		Title:     digStr(src, "videoDetails", "title"),
		Url:       "https://www.youtube.com/watch?v=" + videoID,
		Thumbnail: pickYTPlayerThumb(src),
		Channel:   digStr(src, "videoDetails", "author"),
		Duration:  atoi(digStr(src, "videoDetails", "lengthSeconds")),
		Views:     digStr(src, "videoDetails", "viewCount"),
		Platform:  utils.YouTube,
	}
}

func extractVideoID(u string) string {
	m := videoIDRe1.FindStringSubmatch(u)
	if len(m) > 1 {
		return m[1]
	}
	m2 := videoIDRe2.FindStringSubmatch(u)
	if len(m2) > 1 {
		return m2[1]
	}
	return ""
}

func extractPlaylistID(u string) string {
	m0 := playlistIDRe1.FindStringSubmatch(u)
	if len(m0) > 1 {
		return m0[1]
	}
	m := playlistIDRe2.FindStringSubmatch(u)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func GetYouTubeVideo(ctx context.Context, videoID string) (utils.PlatformTracks, error) {
	payload := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    "WEB",
				"clientVersion": "2.20240229.01.00",
			},
		},
		"videoId": videoID,
	}
	var resp map[string]any
	endpoint := "https://www.youtube.com/youtubei/v1/player?key=AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return utils.PlatformTracks{}, err
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return utils.PlatformTracks{}, err
	}
	video := mapPlayerToTrack(resp)
	if video.Id == "" {
		return utils.PlatformTracks{}, errors.New("video not found")
	}
	return utils.PlatformTracks{Results: []utils.MusicTrack{video}}, nil
}

func GetYouTubePlaylist(ctx context.Context, playlistID string) (utils.PlatformTracks, error) {
	payload := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    "WEB",
				"clientVersion": "2.20240229.01.00",
			},
		},
		"browseId": "VL" + playlistID,
	}
	var resp map[string]any
	endpoint := "https://www.youtube.com/youtubei/v1/browse?key=AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return utils.PlatformTracks{}, err
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return utils.PlatformTracks{}, err
	}
	videos := extractPlaylistVideos(resp)
	out := make([]utils.MusicTrack, 0, len(videos))
	for _, v := range videos {
		out = append(out, mapYTVideo(v))
	}
	return utils.PlatformTracks{Results: out}, nil
}

func GetYouTubeMixPlaylist(ctx context.Context, playlistID string) (utils.PlatformTracks, error) {
	payload := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    "WEB",
				"clientVersion": "2.20240229.01.00",
			},
		},
		"playlistId": playlistID,
	}
	var resp map[string]any
	endpoint := "https://www.youtube.com/youtubei/v1/next?key=AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return utils.PlatformTracks{}, err
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return utils.PlatformTracks{}, err
	}
	videos := extractMixPlaylistVideos(resp)
	out := make([]utils.MusicTrack, 0, len(videos))
	for _, v := range videos {
		out = append(out, mapMixVideo(v))
	}
	return utils.PlatformTracks{Results: out}, nil
}
