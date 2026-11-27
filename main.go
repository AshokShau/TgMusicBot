/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package main

import (
	"ashokshau/tgmusic/internal/bot"
	"ashokshau/tgmusic/internal/calls"
	"ashokshau/tgmusic/internal/cleanup"
	"ashokshau/tgmusic/internal/config"
	"ashokshau/tgmusic/internal/db"
	"ashokshau/tgmusic/internal/disk"
	"ashokshau/tgmusic/internal/downloader"
	"ashokshau/tgmusic/ntgcalls"
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"time"

	"github.com/AshokShau/gotdbot"
	"github.com/AshokShau/gotdbot/logger"
)

//go:generate go run github.com/AshokShau/gotdbot/scripts/tools

var log = logger.New()

func main() {
	if err := config.LoadEnv(); err != nil {
		panic(err)
	}

	downloadsDir, err := filepath.Abs(config.DownloadsDir)
	if err != nil {
		panic(err)
	}
	diskContext, stopDisk := context.WithCancel(context.Background())
	defer stopDisk()
	disk.Start(diskContext, downloadsDir)

	if err := db.InitDatabase(); err != nil {
		panic("failed to connect database: " + err.Error())
	}
	defer func() {
		if err := db.Instance.Close(); err != nil {
			log.Errorf("close database connection: %v", err)
		}
	}()
	defer func() {
		calls.Calls.StopAllClients()
		_ = os.Remove(config.DownloadsDir)
	}()

	startedAssistants := 0
	for _, session := range config.AssistantSessions {
		started, err := calls.Calls.StartClient(config.ApiId, config.ApiHash, session.Value, session.EnvName)
		if err != nil {
			panic(fmt.Errorf("failed to start assistant %s: %w", session.EnvName, err))
		}
		if started {
			startedAssistants++
		}
	}
	if startedAssistants == 0 {
		panic("no assistant could start; all configured assistants are frozen")
	}

	tdDir := "td"
	_ = os.Remove(tdDir)
	manager := gotdbot.NewClientManager("")
	clientConfig := gotdbot.DefaultClientConfig()
	clientConfig.AutoRetry = &gotdbot.AutoRetry{
		ChatNotFound: true,
		MaxFloodWait: 5 * time.Second,
	}

	clientConfig.DatabaseDirectory = tdDir
	clientConfig.ParseMode = gotdbot.ParseModeHTML
	clientConfig.FilesDirectory = filepath.Join(downloadsDir, "main")
	client, err := manager.RegisterClient(config.ApiId, config.ApiHash, config.Token, clientConfig)
	if err != nil {
		panic("failed to register client: " + err.Error())
	}

	if config.DlBotToken != "" {
		dlClientConfig := gotdbot.DefaultClientConfig()
		dlClientConfig.AutoRetry = &gotdbot.AutoRetry{
			ChatNotFound: true,
		}

		dlClientConfig.DatabaseDirectory = tdDir + "_dl"
		dlClientConfig.FilesDirectory = filepath.Join(downloadsDir, "dl")
		_ = os.Remove(dlClientConfig.DatabaseDirectory)

		dlClient, err := manager.RegisterClient(config.ApiId, config.ApiHash, config.DlBotToken, dlClientConfig)
		if err != nil {
			client.Logger.Warnf("failed to register dl client: %s", err.Error())
			downloader.DlBot = client
		} else {
			downloader.DlBot = dlClient
			dlClient.Logger.Infof("dl client registered")
		}
	}

	calls.Calls.RegisterHandlers(client)
	cleanupScheduler, stopCleanup := cleanup.Start(client)
	defer stopCleanup()
	bot.LoadModules(client, cleanupScheduler.CancelChat)
	msg := fmt.Sprintf("Bot started\nNtgCalls %s", ntgcalls.Version())
	client.Logger.Info(msg)
	if _, err := client.SendMessage(config.LoggerId, msg, nil); err != nil {
		log.Warnf("send startup message to logger chat: %v", err)
	}
	manager.Idle()
	client.Logger.Info("The bot is shutting down...")
}
