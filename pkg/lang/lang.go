package lang

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Laky-64/gologging"
)

var translations = make(map[string]map[string]string)

func LoadTranslations() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	execDir := filepath.Dir(execPath)

	localePath := filepath.Join(execDir, "pkg/lang/locale")
	if _, err := os.Stat(localePath); os.IsNotExist(err) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		localePath = filepath.Join(cwd, "pkg/lang/locale")
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
			gologging.InfoF("Loaded language: %s", langCode)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
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
