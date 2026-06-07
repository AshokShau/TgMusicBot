/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package config

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	cookiesDr        = "src/cookies"
	cookieFetchTimeout = 30 * time.Second
	cookieRefreshInterval = time.Hour
)

var cookieHTTPClient = &http.Client{Timeout: cookieFetchTimeout}

// fetchContent downloads content from Pastebin or Batbin.
// It takes a URL as input.
// It returns the content of the URL as a string and an error if any.
func fetchContent(url string) (string, error) {
	parts := strings.Split(strings.Trim(url, "/"), "/")
	id := parts[len(parts)-1]

	var rawURL string
	if strings.Contains(url, "pastebin.com") {
		rawURL = fmt.Sprintf("https://pastebin.com/raw/%s", id)
	} else if strings.Contains(url, "batbin.me") {
		rawURL = fmt.Sprintf("https://batbin.me/raw/%s", id)
	} else {
		rawURL = url
	}

	resp, err := cookieHTTPClient.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to GET %s: %w", rawURL, err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d for %s", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body from %s: %w", rawURL, err)
	}

	return string(body), nil
}

// saveContent saves content to a file in the cookies directory and returns the file path.
// It takes a URL and content as input.
// It returns the file path and an error if any.
func saveContent(url, content string) (string, error) {
	parts := strings.Split(strings.Trim(url, "/"), "/")
	filename := parts[len(parts)-1]
	if filename == "" {
		filename = "file_" + strings.ReplaceAll(strings.Split(strings.ReplaceAll(url, "/", "_"), "?")[0], "#", "")
	}
	filename += ".txt"

	filePath := filepath.Join(cookiesDr, filename)
	// #nosec G304
	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	if _, err := f.WriteString(content); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return filePath, nil
}

// refreshCookies downloads all URLs and atomically replaces Conf.CookiesPath.
// It takes a slice of URLs as input.
func refreshCookies(urls []string) {
	paths := make([]string, 0, len(urls))
	for _, url := range urls {
		content, err := fetchContent(url)
		if err != nil {
			slog.Info("Error fetching cookies from", "url", url, "error", err)
			continue
		}

		path, err := saveContent(url, content)
		if err != nil {
			slog.Info("Error saving cookies for", "url", url, "error", err)
			continue
		}

		paths = append(paths, path)
	}

	Conf.SetCookiesPath(paths)
	slog.Info("Refreshed cookies", "count", len(paths), "total_urls", len(urls))
}

// startCookieRefresher runs an initial fetch and then refreshes on a fixed interval.
// It exits when there are no URLs configured.
func startCookieRefresher(urls []string) {
	if len(urls) == 0 {
		return
	}
	refreshCookies(urls)

	ticker := time.NewTicker(cookieRefreshInterval)
	defer ticker.Stop()
	for range ticker.C {
		refreshCookies(urls)
	}
}
