package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var tmpDir = "src/cookies"

// fetchContent downloads content from Pastebin or Batbin.
// It takes a URL as input.
// It returns the content of the URL as a string and an error if any.
func fetchContent(url string) (string, error) {
	parts := strings.Split(strings.Trim(url, "/"), "/")
	id := parts[len(parts)-1]

	var rawURL string
	if strings.Contains(url, "pastebin.com") {
		rawURL = fmt.Sprintf("https://pastebin.com/raw/%s", id)
	} else {
		rawURL = fmt.Sprintf("https://batbin.me/raw/%s", id)
	}

	resp, err := http.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d for %s", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body from %s: %w", rawURL, err)
	}

	return string(body), nil
}

// saveContent saves content to a file in /tmp and returns the file path.
// It takes a URL and content as input.
// It returns the file path and an error if any.
func saveContent(url, content string) (string, error) {
	parts := strings.Split(strings.Trim(url, "/"), "/")
	filename := parts[len(parts)-1]
	if filename == "" {
		filename = "file_" + strings.ReplaceAll(strings.Split(strings.ReplaceAll(url, "/", "_"), "?")[0], "#", "")
	}
	filename += ".txt"

	filePath := filepath.Join(tmpDir, filename)
	// #nosec G304
	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return filePath, nil
}

// saveAllCookies downloads all URLs and stores paths in Conf.CookiesPath.
// It takes a slice of URLs as input.
func saveAllCookies(urls []string) {
	for _, url := range urls {
		content, err := fetchContent(url)
		if err != nil {
			fmt.Println("Error fetching:", err)
			continue
		}

		path, err := saveContent(url, content)
		if err != nil {
			fmt.Println("Error saving:", err)
			continue
		}

		Conf.CookiesPath = append(Conf.CookiesPath, path)
	}
}
