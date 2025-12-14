/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// BotConfig holds the configuration for the bot.
type BotConfig struct {
	ApiId             int32    // ApiId is the Telegram API ID.
	ApiHash           string   // ApiHash is the Telegram API hash.
	Token             string   // Token is the bot token.
	SessionStrings    []string // SessionStrings is a list of pyrogram/telethon/gogram session strings.
	SessionType       string   // SessionType is the type of session (pyrogram/telethon/gogram).
	MongoUri          string   // MongoUri is the MongoDB connection string.
	DbName            string   // DbName is the name of the database.
	ApiUrl            string   // ApiUrl is the URL of the API.
	ApiKey            string   // ApiKey is the API key.
	OwnerId           int64    // OwnerId is the user ID of the bot owner.
	LoggerId          int64    // LoggerId is the group ID of the bot logger.
	Proxy             string   // Proxy is the proxy URL for the bot.
	DefaultService    string   // DefaultService is the default search platform.
	MaxFileSize       int64    // MaxFileSize is the maximum file size for downloads.
	SongDurationLimit int64    // SongDurationLimit is the maximum duration of a song in seconds.
	DownloadsDir      string   // DownloadsDir is the directory where downloads are stored.
	OwnerUsername     string   // OwnerUsername is the username of the bot owner.
	SupportGroup      string   // SupportGroup is the Telegram group link.
	SupportChannel    string   // SupportChannel is the Telegram channel link.
	DEVS              []int64  // DEVS is a list of developer user IDs.
	CookiesPath       []string // CookiesPath is a list of paths to cookies files.
	cookiesUrl        []string // cookiesUrl is a list of URLs to cookies files.
	Port              string
}

// getSessionStrings gets session strings from environment variable with prefix
func getSessionStrings(prefix string, max int) []string {
	var sessions []string
	for i := 1; i <= max; i++ {
		key := fmt.Sprintf("%s%d", prefix, i)
		if session := os.Getenv(key); session != "" {
			sessions = append(sessions, session)
		}
	}

	// Also check for non-numbered version
	if session := os.Getenv(prefix); session != "" {
		sessions = append(sessions, session)
	}

	return sessions
}

// getEnvStr gets environment variable with default value
func getEnvStr(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvInt32 gets environment variable as int32 with default value
func getEnvInt32(key string, defaultValue int32) int32 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	if val, err := strconv.ParseInt(value, 10, 32); err == nil {
		return int32(val)
	}
	return defaultValue
}

// getEnvInt64 gets environment variable as int64 with default value
func getEnvInt64(key string) int64 {
	value := os.Getenv(key)
	if value == "" {
		return 0
	}
	if val, err := strconv.ParseInt(value, 10, 64); err == nil {
		return val
	}
	return 0
}

// containsInt checks if a slice contains a specific int64 value
func containsInt(slice []int64, val int64) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// processCookieURLs processes comma-separated cookie URLs
func processCookieURLs(urls string) []string {
	if urls == "" {
		return nil
	}
	var result []string
	for _, url := range strings.Split(urls, ",") {
		url = strings.TrimSpace(url)
		if url != "" {
			result = append(result, url)
		}
	}
	return result
}

// validate validates the configuration
func (c *BotConfig) validate() error {
	required := []struct {
		name  string
		value string
		check func() bool
	}{
		{"API_ID", fmt.Sprintf("%d", c.ApiId), func() bool { return c.ApiId > 0 }},
		{"API_HASH", c.ApiHash, func() bool { return c.ApiHash != "" }},
		{"TOKEN", c.Token, func() bool { return c.Token != "" }},
		{"MONGO_URI", c.MongoUri, func() bool { return c.MongoUri != "" }},
		{"OWNER_ID", fmt.Sprintf("%d", c.OwnerId), func() bool { return c.OwnerId > 0 }},
	}

	var missing []string
	for _, req := range required {
		if !req.check() {
			missing = append(missing, req.name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	if len(c.SessionStrings) == 0 {
		return fmt.Errorf("at least one session string (STRING1–10) is required")
	}

	if c.MaxFileSize <= 0 {
		c.MaxFileSize = 500 * 1024 * 1024 // 500MB default
	}

	if c.SongDurationLimit <= 0 {
		c.SongDurationLimit = 3600 // 1 hour default
	}

	if !isValidService(c.DefaultService) {
		c.DefaultService = "youtube"
		log.Printf("Invalid DEFAULT_SERVICE '%s', defaulting to 'youtube'", c.DefaultService)
	}

	return nil
}

// isValidService checks if the service is valid
func isValidService(service string) bool {
	validServices := map[string]bool{
		"youtube": true,
		"spotify": true,
	}
	return validServices[strings.ToLower(service)]
}
