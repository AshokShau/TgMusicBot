package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Laky-64/gologging"
	"github.com/joho/godotenv"
)

// BotConfig holds the configuration for the bot.
type BotConfig struct {
	ApiId          int32    // ApiId is the Telegram API ID.
	ApiHash        string   // ApiHash is the Telegram API hash.
	Token          string   // Token is the bot token.
	SessionStrings []string // SessionStrings is a list of pyrogram session strings.
	MongoUri       string   // MongoUri is the MongoDB connection string.
	DbName         string   // DbName is the name of the database.
	ApiUrl         string   // ApiUrl is the URL of the API.
	ApiKey         string   // ApiKey is the API key.
	OwnerId        int64    // OwnerId is the user ID of the bot owner.
	LoggerId       int64    // LoggerId is the group ID of the bot logger.
	Proxy          string   // Proxy is the proxy URL for the bot.
	DefaultService string   // DefaultService is the default search platform.
	MinMemberCount int64    // MinMemberCount is the minimum number of members required to use the bot.
	MaxFileSize    int64    // MaxFileSize is the maximum file size for downloads.
	DownloadsDir   string   // DownloadsDir is the directory where downloads are stored.
	SupportGroup   string   // SupportGroup is the Telegram group link.
	SupportChannel string   // SupportChannel is the Telegram channel link.
	DEVS           []int64  // DEVS is a list of developer user IDs.
	CookiesPath    []string // CookiesPath is a list of paths to cookies files.
	cookiesUrl     []string // cookiesUrl is a list of URLs to cookies files.
}

// Conf is the global configuration for the bot.
var Conf *BotConfig

// LoadConfig loads the configuration from environment variables and sets the global Conf.
// It also validates the configuration and saves cookies if provided.
func LoadConfig() error {
	_ = godotenv.Load()

	Conf = &BotConfig{
		ApiId:          getEnvInt32("API_ID", 0),
		ApiHash:        os.Getenv("API_HASH"),
		Token:          os.Getenv("TOKEN"),
		SessionStrings: getSessionStrings("STRING", 10),
		MongoUri:       os.Getenv("MONGO_URI"),
		DbName:         getEnvStr("DB_NAME", "MusicBot"),
		ApiUrl:         getEnvStr("API_URL", "https://tgmusic.fallenapi.fun"),
		ApiKey:         os.Getenv("API_KEY"),
		OwnerId:        getEnvInt64("OWNER_ID", 5938660179),
		LoggerId:       getEnvInt64("LOGGER_ID", -1002166934878),
		Proxy:          os.Getenv("PROXY"),
		DefaultService: strings.ToLower(getEnvStr("DEFAULT_SERVICE", "youtube")),
		MinMemberCount: getEnvInt64("MIN_MEMBER_COUNT", 50),
		MaxFileSize:    getEnvInt64("MAX_FILE_SIZE", 500*1024*1024),
		DownloadsDir:   getEnvStr("DOWNLOADS_DIR", "downloads"),
		SupportGroup:   getEnvStr("SUPPORT_GROUP", "https://t.me/GuardxSupport"),
		SupportChannel: getEnvStr("SUPPORT_CHANNEL", "https://t.me/FallenProjects"),
		cookiesUrl:     processCookieURLs(os.Getenv("COOKIES_URL")),
	}

	// Parse DEVS list
	devsEnv := os.Getenv("DEVS")
	if devsEnv != "" {
		for _, idStr := range strings.Fields(devsEnv) {
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				Conf.DEVS = append(Conf.DEVS, id)
			}
		}
	}
	if Conf.OwnerId != 0 && !containsInt(Conf.DEVS, Conf.OwnerId) {
		Conf.DEVS = append(Conf.DEVS, Conf.OwnerId)
	}

	if err := Conf.validate(); err != nil {
		return err
	}

	if len(Conf.cookiesUrl) > 0 {
		if err := os.MkdirAll(tmpDir, 0750); err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}

		gologging.InfoF("Saving cookies...")
		go saveAllCookies(Conf.cookiesUrl)
	}
	return nil
}
