/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ashokshau/tgmusic/src"
	"ashokshau/tgmusic/src/config"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/lang"
	"ashokshau/tgmusic/src/vc"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// handleFlood manages flood wait errors by pausing execution for the specified duration.
// It returns true if a flood wait error is handled, and false otherwise.
func handleFlood(err error) bool {
	if wait := tg.GetFloodWait(err); wait > 0 {
		log.Printf("A flood wait has been detected. Sleeping for %ds.", wait)
		time.Sleep(time.Duration(wait) * time.Second)
		return true
	}
	return false
}

//go:generate go run setup_ntgcalls.go static

// main serves as the entry point for the application.
// It initializes the configuration, database, and Telegram client, then starts the bot and waits for a shutdown signal.
func main() {
	if err := config.LoadConfig(); err != nil {
		panic(err)
	}

	err := lang.LoadTranslations()
	if err != nil {
		panic(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := tg.NewClientConfigBuilder(config.Conf.ApiId, config.Conf.ApiHash).
		WithSession("bot.dat").
		WithFloodHandler(handleFlood).
		WithLogLevel(2).
		Build()

	client, err := tg.NewClient(cfg)
	if err != nil {
		panic(err)
	}

	client.Log.SetColor(true)
	_, err = client.Conn()
	if err != nil {
		panic(err)
	}

	err = client.LoginBot(config.Conf.Token)
	if err != nil {
		panic(err)
	}

	if err := db.InitDatabase(ctx); err != nil {
		panic(err)
	}

	err = pkg.Init(client)
	if err != nil {
		panic(err)
		return
	}

	client.Log.Info("The bot is running as @%s.", client.Me().Username)
	_, _ = client.SendMessage(config.Conf.LoggerId, "The bot has started!")

	<-ctx.Done()
	// client.Idle()
	client.Log.Info("The bot is shutting down...")
	vc.Calls.StopAllClients()
	_ = client.Stop()
}
