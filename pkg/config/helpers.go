package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// getEnvStr retrieves a string from an environment variable or returns a default value.
// It takes the environment variable key and a default string as input.
// It returns the value of the environment variable if it exists, otherwise it returns the default value.
func getEnvStr(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

// getEnvInt64 retrieves an int64 from an environment variable or returns a default value.
// It takes the environment variable key and a default int64 as input.
// It returns the value of the environment variable if it exists and is a valid int64, otherwise it returns the default value.
func getEnvInt64(key string, def int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return def
	}
	return i
}

// getEnvInt32 retrieves an int32 from an environment variable or returns a default value.
// It takes the environment variable key and a default int32 as input.
// It returns the value of the environment variable if it exists and is a valid int32, otherwise it returns the default value.
func getEnvInt32(key string, def int32) int32 {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	i, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return def
	}
	return int32(i)
}

// getEnvBool retrieves a boolean from an environment variable or returns a default value.
// It takes the environment variable key and a default boolean as input.
// It returns the value of the environment variable if it exists and is a valid boolean, otherwise it returns the default value.
func getEnvBool(key string, def bool) bool {
	val := strings.ToLower(os.Getenv(key))
	if val == "" {
		return def
	}
	return val == "true"
}

// getSessionStrings retrieves a list of session strings from environment variables.
// It takes a prefix and a count as input.
// It returns a slice of strings containing the session strings.
func getSessionStrings(prefix string, count int) []string {
	var sessions []string
	for i := 1; i <= count; i++ {
		if s := strings.TrimSpace(os.Getenv(fmt.Sprintf("%s%d", prefix, i))); s != "" {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// processCookieURLs processes a string of cookie URLs into a slice of strings.
// It takes a string of cookie URLs as input.
// It returns a slice of strings containing the individual cookie URLs.
func processCookieURLs(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Fields(strings.ReplaceAll(value, ",", " "))
	var urls []string
	for _, u := range parts {
		if u != "" {
			urls = append(urls, strings.TrimSpace(u))
		}
	}
	return urls
}

// containsInt checks if a slice of int64 contains a specific value.
// It takes a slice of int64 and an int64 as input.
// It returns true if the slice contains the value, otherwise it returns false.
func containsInt(list []int64, x int64) bool {
	for _, v := range list {
		if v == x {
			return true
		}
	}
	return false
}

// validate checks if the bot configuration is valid.
// It returns an error if the configuration is invalid, otherwise it returns nil.
func (c *BotConfig) validate() error {
	var missing []string
	if c.ApiId == 0 {
		missing = append(missing, "API_ID")
	}
	if c.ApiHash == "" {
		missing = append(missing, "API_HASH")
	}
	if c.Token == "" {
		missing = append(missing, "TOKEN")
	}
	if c.MongoUri == "" {
		missing = append(missing, "MONGO_URI")
	}
	if c.LoggerId == 0 {
		missing = append(missing, "LOGGER_ID")
	}
	if c.DbName == "" {
		missing = append(missing, "DB_NAME")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}

	if len(c.SessionStrings) == 0 {
		return fmt.Errorf("at least one session string (STRING1–10) is required")
	}

	if err := os.MkdirAll(c.DownloadsDir, 0750); err != nil {
		return fmt.Errorf("failed to create downloads dir: %v", err)
	}

	return nil
}
