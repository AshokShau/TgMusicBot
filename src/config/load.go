/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// loadEnvFiles loads environment variables from multiple files
func loadEnvFiles(paths ...string) error {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			if err := loadSingleEnvFile(path); err != nil {
				return fmt.Errorf("error loading %s: %w", path, err)
			}
		}
	}
	return nil
}

// loadSingleEnvFile loads environment variables from a single file
func loadSingleEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentKey string
	var currentValue strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if currentKey != "" && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
			currentValue.WriteString("\n" + strings.TrimSpace(line))
			continue
		}

		if currentKey != "" {
			value := strings.TrimSpace(currentValue.String())
			os.Setenv(currentKey, unquoteValue(value))
			currentKey = ""
			currentValue.Reset()
		}

		idx := strings.Index(line, "=")
		if idx == -1 {
			log.Printf("Skipping invalid line in %s: %s", path, line)
			continue
		}

		key := strings.TrimSpace(line[:idx])
		valuePart := strings.TrimSpace(line[idx+1:])
		if commentIdx := strings.Index(valuePart, " #"); commentIdx != -1 {
			valuePart = strings.TrimSpace(valuePart[:commentIdx])
		}

		if strings.HasSuffix(valuePart, "\\") {
			currentKey = key
			currentValue.WriteString(strings.TrimSuffix(valuePart, "\\"))
			continue
		}

		os.Setenv(key, unquoteValue(valuePart))
	}

	if currentKey != "" {
		value := strings.TrimSpace(currentValue.String())
		os.Setenv(currentKey, unquoteValue(value))
	}

	return scanner.Err()
}

// unquoteValue removes surrounding quotes from values
func unquoteValue(value string) string {
	value = strings.TrimSpace(value)

	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}

	return value
}
