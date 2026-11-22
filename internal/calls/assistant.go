/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package calls

import (
	"ashokshau/tgmusic/internal/calls/sessions"
	"ashokshau/tgmusic/internal/config"
	"ashokshau/tgmusic/internal/db"
	"ashokshau/tgmusic/ntgcalls"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"

	tg "github.com/xAmarnath/gogram/telegram"
)

type pendingConnection struct {
	MediaDesc ntgcalls.MediaDescription
	Payload   string
}

type AssistantAccount struct {
	App     *tg.Client
	binding *ntgcalls.Client
	self    *tg.UserObj

	mu                 sync.RWMutex
	mutedByAdmin       []int64
	presentations      []int64
	inputGroupCalls    map[int64]tg.InputGroupCall
	pendingConnections map[int64]*pendingConnection
	waitConnect        map[int64]chan error
}

func (a *AssistantAccount) Close() {
	if a.binding != nil {
		a.binding.Free()
	}
}

func (c *TelegramCalls) newAssistantAccount(app *tg.Client) (*AssistantAccount, error) {
	binding := ntgcalls.NTgCalls()

	a := &AssistantAccount{
		App:                app,
		binding:            binding,
		inputGroupCalls:    make(map[int64]tg.InputGroupCall),
		pendingConnections: make(map[int64]*pendingConnection),
		waitConnect:        make(map[int64]chan error),
	}

	self, err := app.GetMe()
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant account details: %w", err)
	}

	a.self = self
	c.handleUpdates(a)
	return a, nil
}

// StartClient starts an assistant and reports whether it can be used for calls.
func (c *TelegramCalls) StartClient(apiID int32, apiHash, stringSession, name string) (bool, error) {
	var sess *tg.Session
	var err error

	clientConfig := tg.ClientConfig{
		AppID:         apiID,
		AppHash:       apiHash,
		MemorySession: true,
		FloodHandler:  handleFlood,
		LogLevel:      tg.InfoLevel,
		SessionName:   name,
	}

	switch config.SessionType {
	case "telethon":
		sess, err = sessions.DecodeTelethonSessionString(stringSession)
		if err != nil {
			return false, fmt.Errorf("failed to decode telethon session string: %w", err)
		}
		clientConfig.StringSession = sess.Encode()
	case "pyrogram":
		sess, err = sessions.DecodePyrogramSessionString(stringSession)
		if err != nil {
			return false, fmt.Errorf("failed to decode pyrogram session string: %w", err)
		}
		clientConfig.StringSession = sess.Encode()
	case "gogram":
		clientConfig.StringSession = stringSession
	default:
		return false, fmt.Errorf("unsupported session type: %s", config.SessionType)
	}

	mtProto, err := tg.NewClient(clientConfig)
	if err != nil {
		return false, fmt.Errorf("failed to create the MTProto client: %w", err)
	}

	if err = mtProto.Start(); err != nil {
		_ = mtProto.Stop()
		return false, fmt.Errorf("failed to start the client: %w", err)
	}

	me := mtProto.Me()
	if me.Bot {
		_ = mtProto.Stop()
		return false, fmt.Errorf("the client is a bot")
	}

	appConfig, err := mtProto.HelpGetAppConfig(0)
	if err != nil {
		logger.Warn("[TelegramCalls] failed to fetch app config", "id", me.ID, "username", me.Username, "error", err)
	} else {
		isFreeze := false
		if cfg, ok := appConfig.(*tg.HelpAppConfigObj); ok {
			if cfgObj, ok := cfg.Config.(*tg.JsonObject); ok {
				for _, entry := range cfgObj.Value {
					if entry != nil && entry.Key == "freeze_since_date" {
						isFreeze = true
						break
					}
				}
			}
		}

		if isFreeze {
			logger.Warn("[TelegramCalls] Skipping frozen assistant", "session", name, "id", me.ID, "username", me.Username)
			_ = mtProto.Stop()
			return false, nil
		}
	}

	acc, err := c.newAssistantAccount(mtProto)
	if err != nil {
		_ = mtProto.Stop()
		return false, fmt.Errorf("failed to create the assistant instance: %w", err)
	}

	c.mu.Lock()
	c.accounts = append(c.accounts, acc)
	clientIndex := len(c.accounts) - 1
	c.mu.Unlock()

	logger.Info("[TelegramCalls] Client started", "client", fmt.Sprintf("client%d", clientIndex), "id", me.ID, "username", me.Username)
	return true, nil
}

func (c *TelegramCalls) StopAllClients() {
	c.mu.RLock()
	accounts := make([]*AssistantAccount, len(c.accounts))
	copy(accounts, c.accounts)
	c.mu.RUnlock()

	for _, acc := range accounts {
		acc.Close()
		logger.Infof("stopping assistant client %d", acc.self.ID)
		_ = acc.App.Stop()
	}
}

func (c *TelegramCalls) accountIndexFor(acc *AssistantAccount) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for idx, account := range c.accounts {
		if account == acc {
			return idx
		}
	}
	return -1
}

func (c *TelegramCalls) getAccountIndex(chatID int64) (int, error) {
	c.mu.RLock()
	totalAccounts := len(c.accounts)
	c.mu.RUnlock()

	if totalAccounts == 0 {
		return -1, fmt.Errorf("no accounts are available")
	}

	assignedIndex, err := db.Instance.GetAssistant(chatID)
	if err != nil {
		return -1, fmt.Errorf("get assistant for chat %d: %w", chatID, err)
	}

	if assignedIndex >= 0 && assignedIndex < totalAccounts {
		return assignedIndex, nil
	}
	if assignedIndex != -1 {
		logger.Warnf("stored assistant index %d for chat %d is unavailable with %d active assistants; reassigning", assignedIndex, chatID, totalAccounts)
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(totalAccounts)))
	if err != nil {
		return -1, fmt.Errorf("select assistant for chat %d: %w", chatID, err)
	}

	newAccountIndex := int(n.Int64())
	if chatID != 0 {
		assigned, err := db.Instance.AssignAssistant(chatID, newAccountIndex, totalAccounts)
		if err != nil {
			return -1, fmt.Errorf("assign assistant for chat %d: %w", chatID, err)
		}
		if assigned < 0 || assigned >= totalAccounts {
			logger.Warnf("assistant assignment %d for chat %d is still unavailable; replacing with %d", assigned, chatID, newAccountIndex)
			if err := db.Instance.SetAssistant(chatID, newAccountIndex); err != nil {
				return -1, fmt.Errorf("replace unavailable assistant for chat %d: %w", chatID, err)
			}
		} else {
			newAccountIndex = assigned
		}
	}

	return newAccountIndex, nil
}

func (c *TelegramCalls) GetAccount(chatID int64) (*AssistantAccount, int, error) {
	accountIndex, err := c.getAccountIndex(chatID)
	if err != nil {
		return nil, -1, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if accountIndex < 0 || accountIndex >= len(c.accounts) {
		return nil, -1, fmt.Errorf("no assistant account was found for index %d", accountIndex)
	}
	return c.accounts[accountIndex], accountIndex, nil
}

func (c *TelegramCalls) nextUntried(tried map[int]bool) (*AssistantAccount, int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for i, call := range c.accounts {
		if !tried[i] {
			return call, i, nil
		}
	}
	return nil, -1, errors.New("no untried assistants remain")
}
