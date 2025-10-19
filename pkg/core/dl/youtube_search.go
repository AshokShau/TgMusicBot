package dl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
)

// searchYouTube scrapes YouTube results page
func searchYouTube(query string) ([]cache.MusicTrack, error) {
	query = strings.ReplaceAll(query, " ", "+")
	url := "https://www.youtube.com/results?search_query=" + query
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	re := regexp.MustCompile(`var ytInitialData = (.*?);\s*</script>`)
	match := re.FindSubmatch(body)
	if len(match) < 2 {
		return nil, fmt.Errorf("ytInitialData not found")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(match[1], &data); err != nil {
		return nil, err
	}

	// Navigate nested fields
	contents := dig(data, "contents", "twoColumnSearchResultsRenderer",
		"primaryContents", "sectionListRenderer", "contents")

	if contents == nil {
		return nil, fmt.Errorf("no contents")
	}

	var tracks []cache.MusicTrack
	parseSearchResults(contents, &tracks)

	return tracks, nil
}

// Recursively find items
func parseSearchResults(node interface{}, tracks *[]cache.MusicTrack) {
	switch v := node.(type) {
	case []interface{}:
		for _, item := range v {
			parseSearchResults(item, tracks)
		}
	case map[string]interface{}:
		if vid, ok := dig(v, "videoRenderer").(map[string]interface{}); ok {
			id := safeString(vid["videoId"])
			title := safeString(dig(vid, "title", "runs", 0, "text"))
			thumb := safeString(dig(vid, "thumbnail", "thumbnails", 0, "url"))
			durationText := safeString(dig(vid, "lengthText", "simpleText"))
			duration := parseDuration(durationText)
			*tracks = append(*tracks, cache.MusicTrack{
				URL:      "https://www.youtube.com/watch?v=" + id,
				Name:     title,
				ID:       id,
				Cover:    thumb,
				Duration: duration,
				Platform: "youtube",
			})
		} else {
			for _, child := range v {
				parseSearchResults(child, tracks)
			}
		}
	}
}

// safely dig into nested JSON
func dig(m interface{}, path ...interface{}) interface{} {
	curr := m
	for _, p := range path {
		switch key := p.(type) {
		case string:
			if mm, ok := curr.(map[string]interface{}); ok {
				curr = mm[key]
			} else {
				return nil
			}
		case int:
			if arr, ok := curr.([]interface{}); ok && len(arr) > key {
				curr = arr[key]
			} else {
				return nil
			}
		}
	}
	return curr
}

// safely cast to string
func safeString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// parse duration like "3:45" -> 225 seconds
func parseDuration(s string) int {
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ":")
	total := 0
	multiplier := 1

	// Process from right to left (seconds → minutes → hours)
	for i := len(parts) - 1; i >= 0; i-- {
		total += atoi(parts[i]) * multiplier
		multiplier *= 60
	}
	return total
}

// atoi converts a string to an integer
func atoi(s string) int {
	var n int
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	return n
}
