/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package main

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src"
	"ashokshau/tgmusic/src/core/dl"
	"ashokshau/tgmusic/src/handlers"
	"ashokshau/tgmusic/src/vc"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"time"

	"github.com/AshokShau/gotdbot"
)

//go:generate go run github.com/AshokShau/gotdbot/scripts/tools

// main serves as the entry point for the application.
func main() {
	go func() {
		if err := http.ListenAndServe("0.0.0.0:"+config.Port, nil); err != nil {
			slog.Info("pprof server error", "error", err)
		}
	}()

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					t := a.Value.Time()
					a.Value = slog.StringValue(t.Format("2006-01-02 15:04:05"))
				}

				if a.Key == slog.SourceKey {
					source := a.Value.Any().(*slog.Source)
					a.Value = slog.StringValue(fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line))
				}

				return a
			},
		}),
	)

	slog.SetDefault(logger)
	tdDir := "database"
	_ = os.Remove(tdDir)
	libPath := "./libtdjson.so.1.8.65"

	manager := gotdbot.NewClientManager(libPath)

	clientConfig := gotdbot.DefaultClientConfig()
	clientConfig.AutoRetry = &gotdbot.AutoRetry{
		ChatNotFound: true,
		MaxFloodWait: 5 * time.Second,
	}

	clientConfig.DatabaseDirectory = tdDir
	client, err := manager.RegisterClient(config.ApiId, config.ApiHash, config.Token, clientConfig)
	if err != nil {
		slog.Error("manager.RegisterClient error", "error", err)
		os.Exit(1)
	}

	if config.DlBotToken != "" {
		dlClientConfig := gotdbot.DefaultClientConfig()
		dlClientConfig.AutoRetry = &gotdbot.AutoRetry{
			ChatNotFound: true,
		}
		dlClientConfig.DatabaseDirectory = tdDir + "_dl"
		_ = os.Remove(dlClientConfig.DatabaseDirectory)

		dlClient, err := manager.RegisterClient(config.ApiId, config.ApiHash, config.DlBotToken, dlClientConfig)
		if err != nil {
			slog.Error("manager.RegisterClient (DL) error", "error", err)
			dl.DlBot = client
		} else {
			dl.DlBot = dlClient
			dlClient.Logger.Info("Download bot registered successfully")
		}
	}

	err = src.Init(client)
	if err != nil {
		panic(err)
	}

	handlers.LoadModules(client)
	_, _ = client.SendTextMessage(config.LoggerId, "The bot has started!", nil)
	manager.Idle()
	client.Logger.Info("The bot is shutting down...")
	vc.Calls.StopAllClients()
}
