/*
 * TgMusicBot - Telegram Music Bot
 * Copyright (c) 2025-2026 Ashok Shau
 *
 * Licensed under GNU GPL v3
 * See https://github.com/AshokShau/TgMusicBot
 */

package config

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	cookiesDr = "src/cookies"

	maxCookieSize = 10 << 20 // 10 MiB

	requestTimeout = 30 * time.Second
)

var cookieHTTPClient = &http.Client{
	Timeout: requestTimeout,
}
var cookiesMu sync.RWMutex

// CookieFiles returns a snapshot of downloaded cookie file paths.
func CookieFiles() []string {
	cookiesMu.RLock()
	defer cookiesMu.RUnlock()
	return append([]string(nil), CookiesPath...)
}

func cookieFailure(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}
	return err
}

// fetchContent downloads content from Pastebin, Batbin, or a direct URL.
func fetchContent(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	downloadURL, err := resolveRawURL(rawURL)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("create cookie request: %w", err)
	}

	req.Header.Set("User-Agent", "TgMusicBot/1.0")

	resp, err := cookieHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request cookie source: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cookie source returned HTTP %s", resp.Status)
	}

	// Prevent unexpectedly large responses from consuming memory.
	reader := io.LimitReader(resp.Body, maxCookieSize+1)

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read cookie response: %w", err)
	}

	if len(body) > maxCookieSize {
		return "", fmt.Errorf("cookie response exceeds maximum size of %d bytes", maxCookieSize)
	}

	return string(body), nil
}

// resolveRawURL converts supported paste services into their raw URLs.
func resolveRawURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid cookie URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}

	host := strings.ToLower(parsed.Hostname())
	path := strings.Trim(parsed.Path, "/")

	switch host {
	case "pastebin.com", "www.pastebin.com":
		if path == "" {
			return "", fmt.Errorf("invalid Pastebin URL")
		}

		id, _, _ := strings.Cut(path, "/")
		return "https://pastebin.com/raw/" + id, nil

	case "batbin.me", "www.batbin.me":
		if path == "" {
			return "", fmt.Errorf("invalid Batbin URL")
		}

		id, _, _ := strings.Cut(path, "/")
		return "https://batbin.me/raw/" + id, nil

	default:
		return rawURL, nil
	}
}

// saveContent saves content to the cookies directory and returns the file path.
func saveContent(sourceURL, content string) (string, error) {
	filename := cookieFilename(sourceURL)

	if err := os.MkdirAll(cookiesDr, 0o750); err != nil {
		return "", fmt.Errorf("create cookies directory: %w", err)
	}

	filePath := filepath.Join(cookiesDr, filename)

	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("write cookie file %q: %w", filePath, err)
	}

	return filePath, nil
}

// cookieFilename creates a safe filename from the source URL.
func cookieFilename(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil {
		if path := strings.Trim(parsed.Path, "/"); path != "" {
			name := filepath.Base(path)
			name = sanitizeFilename(name)

			if name != "" && name != "." && name != ".." {
				return name + ".txt"
			}
		}

		if parsed.Hostname() != "" {
			return sanitizeFilename(parsed.Hostname()) + ".txt"
		}
	}

	return "cookie.txt"
}

// sanitizeFilename removes characters that could make the filename unsafe.
func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)

	var builder strings.Builder
	builder.Grow(len(name))

	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-', r == '_', r == '.':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}

	name = strings.Trim(builder.String(), "._")

	if name == "" {
		return "cookie"
	}

	return name
}

func saveAllCookies(urls []string) {
	if len(urls) == 0 {
		return
	}

	var wg sync.WaitGroup

	for source, rawURL := range urls {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			continue
		}

		wg.Add(1)

		go func(rawURL string, source int) {
			defer wg.Done()

			content, err := fetchContent(rawURL)
			if err != nil {
				log.Warnf("fetch cookies from source %d: %v", source+1, cookieFailure(err))
				return
			}

			path, err := saveContent(rawURL, content)
			if err != nil {
				log.Errorf("save cookies from source %d: %v", source+1, cookieFailure(err))
				return
			}

			cookiesMu.Lock()
			CookiesPath = append(CookiesPath, path)
			cookiesMu.Unlock()

		}(rawURL, source)
	}

	wg.Wait()

	log.Infof("cookie downloads completed: requested=%d saved=%d", len(urls), len(CookieFiles()))
}
