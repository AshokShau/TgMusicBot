/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package main

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src"
	"log"
	"time"

	"net/http"
	_ "net/http/pprof"

	"ashokshau/tgmusic/src/vc"

	tg "github.com/amarnathcjd/gogram/telegram"
)

//go:generate go run setup_ntgcalls.go static

// main serves as the entry point for the application.
func main() {
	if err := config.LoadConfig(); err != nil {
		panic(err)
	}

	go func() {
		if err := http.ListenAndServe("0.0.0.0:"+config.Conf.Port, nil); err != nil {
			log.Println("pprof server error:", err)
		}
	}()

	clientConfig := tg.ClientConfig{
		AppID:        config.Conf.ApiId,
		AppHash:      config.Conf.ApiHash,
		FloodHandler: handleFlood,
		SessionName:  "bot",
		LogLevel:     tg.TraceLevel,
	}

	client, err := tg.NewClient(clientConfig)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	_, err = client.Conn()
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	err = client.LoginBot(config.Conf.Token)
	if err != nil {
		log.Fatalf("failed to login: %v", err)
	}

	err = src.Init(client)
	if err != nil {
		panic(err)
	}

	userName := client.Me().Username
	if userName == "" {
		log.Fatal("failed to get bot username")
	}

	client.Log.Infof("The bot is running as @%s.", userName)
	_, _ = client.SendMessage(config.Conf.LoggerId, "The bot has started!")
	client.Idle()
	log.Println("The bot is shutting down...")
	vc.Calls.StopAllClients()
	_ = client.Stop()
}

// handleFlood manages flood wait errors by pausing execution for the specified duration.
// It returns true if a flood wait error is handled, and false otherwise.
func handleFlood(err error) bool {
	if wait := tg.GetFloodWait(err); wait > 0 {
		log.Printf("A flood wait has been detected. Sleeping for %ds (%v).", wait, err)
		time.Sleep(time.Duration(wait) * time.Second)
		return true
	}
	return false
}
