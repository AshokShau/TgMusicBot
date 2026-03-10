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
	"ashokshau/tgmusic/src/handlers"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"

	"ashokshau/tgmusic/src/vc"

	"github.com/AshokShau/gotdbot"
)

//go:generate go run github.com/AshokShau/gotdbot/scripts/tools@latest

// main serves as the entry point for the application.
func main() {
	if err := config.LoadConfig(); err != nil {
		panic(err)
	}

	go func() {
		if err := http.ListenAndServe("0.0.0.0:"+config.Conf.Port, nil); err != nil {
			slog.Info("pprof server error", "error", err)
		}
	}()

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {

				// Shorter time
				if a.Key == slog.TimeKey {
					t := a.Value.Time()
					a.Value = slog.StringValue(t.Format("2006-01-02 15:04:05"))
				}

				// Short source (file.go:line)
				if a.Key == slog.SourceKey {
					source := a.Value.Any().(*slog.Source)
					a.Value = slog.StringValue(fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line))
				}

				return a
			},
		}),
	)

	slog.SetDefault(logger)

	clientConfig := &gotdbot.ClientConfig{
		LibraryPath: "./libtdjson.so.1.8.62",
		Logger:      logger,
		DispatcherOpts: &gotdbot.DispatcherOpts{
			ErrorHandler: func(c *gotdbot.Client, ctx *gotdbot.Context, err error) error {
				logger.Error("Handler error", "error", err)
				return nil
			},
		},
	}

	client, err := gotdbot.NewClient(config.Conf.ApiId, config.Conf.ApiHash, config.Conf.Token, clientConfig)
	if err != nil {
		slog.Error("gotdbot.NewClient error", "error", err)
		os.Exit(1)
	}

	gotdbot.SetTdlibLogStreamEmpty()
	gotdbot.SetTdlibLogVerbosityLevel(2)

	dispatcher := client.Dispatcher
	if err = client.Start(); err != nil {
		slog.Error("gotdbot.Start() error", "error", err)
		os.Exit(1)
	}
	err = src.Init(client)
	if err != nil {
		panic(err)
	}

	handlers.LoadModules(dispatcher)
	me := client.Me()
	username := ""
	if me.Usernames != nil && len(me.Usernames.ActiveUsernames) > 0 {
		username = me.Usernames.ActiveUsernames[0]
	}

	slog.Info("Bot started as @ (ID: )", "arg1", username, "id", me.Id)
	_, _ = client.SendTextMessage(config.Conf.LoggerId, "The bot has started!", nil)
	client.Idle()
	slog.Info("The bot is shutting down...")
	vc.Calls.StopAllClients()
}
