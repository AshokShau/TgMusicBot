/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package vc

import (
"log/slog"
	"sync"
	"time"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/vc/ubot"

	td "github.com/AshokShau/gotdbot"
	tg "github.com/amarnathcjd/gogram/telegram"
)

var logger = slog.Default()

// TelegramCalls manages the state and operations for voice calls, including userbots and the main bot client.
type TelegramCalls struct {
	mu               sync.RWMutex
	uBContext        map[string]*ubot.Context
	clients          map[string]*tg.Client
	availableClients []string
	clientCounter    int
	bot              *td.Client
	statusCache      *cache.Cache[td.ChatMemberStatus]
	inviteCache      *cache.Cache[string]
}

var (
	instance *TelegramCalls
	once     sync.Once
)

// getCalls returns the singleton instance of the TelegramCalls manager, ensuring that only one instance is created.
func getCalls() *TelegramCalls {
	once.Do(func() {
		instance = &TelegramCalls{
			uBContext:     make(map[string]*ubot.Context),
			clients:       make(map[string]*tg.Client),
			clientCounter: 1,
			statusCache:   cache.NewCache[td.ChatMemberStatus](2 * time.Hour),
			inviteCache:   cache.NewCache[string](2 * time.Hour),
		}
	})
	return instance
}

// Calls is the singleton instance of TelegramCalls, initialized lazily.
var Calls = getCalls()
