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
	"log/slog"
	"os"
	"strconv"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

var (
	ApiId             = getEnvInt32("API_ID", 0)
	ApiHash           = os.Getenv("API_HASH")
	Token             = os.Getenv("TOKEN")
	SessionStrings    = getSessionStrings("STRING", 10)
	SessionType       = getEnv("SESSION_TYPE", "pyrogram")
	MongoUri          = os.Getenv("MONGO_URI")
	DbName            = getEnv("DB_NAME", "Anon")
	ApiUrl            = getEnv("API_URL", "https://api.onegrab.fun")
	ApiKey            = os.Getenv("API_KEY")
	OwnerId           = getEnvInt64("OWNER_ID", 0)
	LoggerId          = getEnvInt64("LOGGER_ID", 0)
	Proxy             = os.Getenv("PROXY")
	DefaultService    = strings.ToLower(getEnv("DEFAULT_SERVICE", "youtube"))
	MaxFileSize       = getEnvInt64("MAX_FILE_SIZE", 500*1024*1024)
	SongDurationLimit = getEnvInt64("SONG_DURATION_LIMIT", 3600)
	DownloadsDir      = getEnv("DOWNLOADS_DIR", "database")
	SupportGroup      = getEnv("SUPPORT_GROUP", "https://t.me/FallenSupport")
	SupportChannel    = getEnv("SUPPORT_CHANNEL", "https://t.me/FallenProjects")
	StartImg          = getEnv("START_IMG", "https://i.pinimg.com/736x/0d/f4/65/0df465d1e98239ecb6283400605fc813.jpg")
	Port              = getEnv("PORT", "6060")
	AutoLeave         = getEnvBool("AUTO_LEAVE", false)

	DEVS        []int64
	CookiesPath []string
	cookiesUrl  = processCookieURLs(os.Getenv("COOKIES_URL"))
)

func init() {
	devsEnv := os.Getenv("DEVS")
	if devsEnv != "" {
		devsEnv = strings.ReplaceAll(devsEnv, "\n", " ")
		devsEnv = strings.ReplaceAll(devsEnv, ",", " ")

		for _, idStr := range strings.Fields(devsEnv) {
			idStr = strings.TrimSpace(idStr)
			if idStr == "" {
				continue
			}
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				DEVS = append(DEVS, id)
			} else {
				slog.Info("Invalid DEV ID", "id", idStr, "error", err)
			}
		}
	}

	if OwnerId != 0 && !containsInt(DEVS, OwnerId) {
		DEVS = append(DEVS, OwnerId)
	}

	if err := validate(); err != nil {
		slog.Error("Configuration validation failed", "error", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(DownloadsDir, 0755); err != nil {
		slog.Error("Failed to create downloads directory", "error", err)
		os.Exit(1)
	}

	if len(cookiesUrl) > 0 {
		if err := os.MkdirAll(cookiesDr, 0750); err != nil {
			slog.Error("Failed to create temp dir for cookies", "error", err)
			os.Exit(1)
		}
		go saveAllCookies(cookiesUrl)
	}
}

// getEnv returns the value of an environment variable or a default value if it is not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt64 returns the value of an environment variable as an int64 or a default value if it is not set
func getEnvInt64(key string, defaultValue int64) int64 {
	if value, err := strconv.ParseInt(os.Getenv(key), 10, 64); err == nil {
		return value
	}
	return defaultValue
}

// getEnvInt32 gets environment variable as int32 with default value
func getEnvInt32(key string, defaultValue int32) int32 {
	if value, err := strconv.ParseInt(os.Getenv(key), 10, 32); err == nil {
		return int32(value)
	}
	return defaultValue
}

// getEnvBool gets environment variable as bool with default value
func getEnvBool(key string, defaultValue bool) bool {
	if val, err := strconv.ParseBool(os.Getenv(key)); err == nil {
		return val
	}
	return defaultValue
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

// containsInt checks if a slice contains a specific int64 value
func containsInt(slice []int64, val int64) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// validate validates the configuration
func validate() error {
	required := []struct {
		name  string
		value string
		check func() bool
	}{
		{"API_ID", fmt.Sprintf("%d", ApiId), func() bool { return ApiId > 0 }},
		{"API_HASH", ApiHash, func() bool { return ApiHash != "" }},
		{"TOKEN", Token, func() bool { return Token != "" }},
		{"MONGO_URI", MongoUri, func() bool { return MongoUri != "" }},
		{"OWNER_ID", fmt.Sprintf("%d", OwnerId), func() bool { return OwnerId > 0 }},
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

	if len(SessionStrings) == 0 {
		return fmt.Errorf("at least one session string (STRING1–10) is required")
	}

	if !isValidService(DefaultService) {
		DefaultService = "youtube"
		slog.Info("Invalid DEFAULT_SERVICE, defaulting to 'youtube'", "Service", DefaultService)
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
