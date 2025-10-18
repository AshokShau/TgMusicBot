package vc

import (
	"sync"
	"tgmusic/pkg/core/cache"
	"tgmusic/pkg/vc/ubot"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// TelegramCalls manages the state and operations for voice calls, including userbots and the main bot client.
type TelegramCalls struct {
	mu               sync.RWMutex
	uBContext        map[string]*ubot.Context
	clients          map[string]*tg.Client
	availableClients []string
	clientCounter    int
	bot              *tg.Client
	statusCache      *cache.Cache[string]
	inviteCache      *cache.Cache[string]
}

var (
	instance *TelegramCalls
	once     sync.Once
)

// GetCalls returns the singleton instance of the TelegramCalls manager, ensuring that only one instance is created.
func GetCalls() *TelegramCalls {
	once.Do(func() {
		instance = &TelegramCalls{
			uBContext:     make(map[string]*ubot.Context),
			clients:       make(map[string]*tg.Client),
			clientCounter: 1,
			statusCache:   cache.NewCache[string](2 * time.Hour),
			inviteCache:   cache.NewCache[string](2 * time.Hour),
		}
	})
	return instance
}

// Calls is the singleton instance of TelegramCalls, initialized lazily.
var Calls = GetCalls()
