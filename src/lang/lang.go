/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package lang

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var translations = make(map[string]map[string]string)

func LoadTranslations() (int, error) {
	execPath, err := os.Executable()
	if err != nil {
		return 0, err
	}
	execDir := filepath.Dir(execPath)

	localePath := filepath.Join(execDir, "locales")
	if _, err := os.Stat(localePath); os.IsNotExist(err) {
		cwd, err := os.Getwd()
		if err != nil {
			return 0, err
		}
		localePath = filepath.Join(cwd, "locales")
	}

	err = filepath.Walk(localePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
			langCode := strings.TrimSuffix(info.Name(), ".json")
			file, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var langMap map[string]string
			if err := json.Unmarshal(file, &langMap); err != nil {
				return err
			}
			translations[langCode] = langMap
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return len(translations), nil
}

func GetString(langCode, key string) string {
	if lang, ok := translations[langCode]; ok {
		if val, ok := lang[key]; ok {
			return val
		}
	}
	// Fallback to English
	if lang, ok := translations["en"]; ok {
		if val, ok := lang[key]; ok {
			return val
		}
	}
	return key
}

func GetAvailableLangs() []string {
	langs := make([]string, 0, len(translations))
	for k := range translations {
		langs = append(langs, k)
	}
	sort.Strings(langs)
	return langs
}

func GetLangDisplayName(langCode string) string {
	if lang, ok := translations[langCode]; ok {
		if val, ok := lang["lang_name"]; ok {
			return val
		}
	}

	return "Unknown"
}
