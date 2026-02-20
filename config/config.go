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
	"log"
	"os"
	"strconv"
	"strings"
)

// Conf is the global configuration for the bot.
var Conf *BotConfig

// LoadConfig loads the configuration from environment variables and sets the global Conf.
func LoadConfig() error {
	envFiles := []string{".env.local", ".env"}
	if err := loadEnvFiles(envFiles...); err != nil {
		log.Printf("Warning loading env files: %v", err)
	}

	Conf = &BotConfig{
		ApiId:             getEnvInt32("API_ID", 0),
		ApiHash:           os.Getenv("API_HASH"),
		Token:             os.Getenv("TOKEN"),
		SessionStrings:    getSessionStrings("STRING", 10),
		SessionType:       getEnvStr("SESSION_TYPE", "pyrogram"),
		MongoUri:          os.Getenv("MONGO_URI"),
		DbName:            getEnvStr("DB_NAME", "MusicBot"),
		ApiUrl:            getEnvStr("API_URL", "https://beta.fallenapi.fun"),
		ApiKey:            os.Getenv("API_KEY"),
		OwnerId:           getEnvInt64("OWNER_ID"),
		LoggerId:          getEnvInt64("LOGGER_ID"),
		Proxy:             os.Getenv("PROXY"),
		DefaultService:    strings.ToLower(getEnvStr("DEFAULT_SERVICE", "youtube")),
		MaxFileSize:       getEnvInt64("MAX_FILE_SIZE"),
		SongDurationLimit: getEnvInt64("SONG_DURATION_LIMIT"),
		DownloadsDir:      getEnvStr("DOWNLOADS_DIR", "downloads"),
		SupportGroup:      getEnvStr("SUPPORT_GROUP", "https://t.me/FallenSupport"),
		SupportChannel:    getEnvStr("SUPPORT_CHANNEL", "https://t.me/FallenProjects"),
		cookiesUrl:        processCookieURLs(os.Getenv("COOKIES_URL")),
		Port:              getEnvStr("PORT", "6060"),
	}

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
				Conf.DEVS = append(Conf.DEVS, id)
			} else {
				log.Printf("Invalid DEV ID '%s': %v", idStr, err)
			}
		}
	}

	if Conf.OwnerId != 0 && !containsInt(Conf.DEVS, Conf.OwnerId) {
		Conf.DEVS = append(Conf.DEVS, Conf.OwnerId)
	}

	if err := Conf.validate(); err != nil {
		return err
	}

	if err := os.MkdirAll(Conf.DownloadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	if len(Conf.cookiesUrl) > 0 {
		if err := os.MkdirAll(cookiesDr, 0750); err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}
		go saveAllCookies(Conf.cookiesUrl)
	}

	return nil
}
