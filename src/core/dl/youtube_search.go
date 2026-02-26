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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func searchYouTube(query string, limit int) ([]utils.MusicTrack, error) {
	endpoint := "https://www.youtube.com/youtubei/v1/search?key=AIzaSyBOti4mM-6x9WDnZIjIeyEU21OpBXqWBgw"

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
