package main

import (
	"time"

	"github.com/AshokShau/TgMusicBot/pkg"
	"github.com/AshokShau/TgMusicBot/pkg/config"
	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/vc"

	_ "net/http"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// handleFlood manages flood wait errors by pausing execution for the specified duration.
// It returns true if a flood wait error is handled, and false otherwise.
func handleFlood(err error) bool {
	if wait := tg.GetFloodWait(err); wait > 0 {
		gologging.InfoF("A flood wait has been detected. Sleeping for %ds.", wait)
		time.Sleep(time.Duration(wait) * time.Second)
		return true
	}
	return false
}

//go:generate go run setup_ntgcalls.go static

// main serves as the entry point for the application.
// It initializes the configuration, database, and Telegram client, then starts the bot and waits for a shutdown signal.
func main() {
	gologging.SetLevel(gologging.InfoLevel)
	gologging.GetLogger("ntgcalls").SetLevel(gologging.InfoLevel)
	gologging.GetLogger("webrtc").SetLevel(gologging.FatalLevel)

	if err := config.LoadConfig(); err != nil {
		gologging.Fatal(err.Error())
	}

	// ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	ctx, cancel := db.Ctx()
	defer cancel()

	cfg := tg.NewClientConfigBuilder(config.Conf.ApiId, config.Conf.ApiHash).
		WithSession("bot.dat").
		WithFloodHandler(handleFlood).
		Build()

	client, err := tg.NewClient(cfg)
	if err != nil {
		gologging.FatalF("Failed to create the client: %v", err)
	}

	_, err = client.Conn()
	if err != nil {
		gologging.FatalF("Failed to connect to Telegram: %v", err)
	}

	err = client.LoginBot(config.Conf.Token)
	if err != nil {
		gologging.FatalF("Failed to log in as the bot: %v", err)
	}

	if err := db.InitDatabase(ctx); err != nil {
		panic(err)
	}

	err = pkg.Init(client)
	if err != nil {
		gologging.FatalF("Failed to initialize the package: %v", err)
		return
	}
	gologging.InfoF("The bot is running as @%s.", client.Me().Username)
	_, _ = client.SendMessage(config.Conf.LoggerId, "The bot has started!")

	// <-ctx.Done()
	client.Idle()
	gologging.InfoF("The bot is shutting down...")
	vc.Calls.StopAllClients()
	_ = client.Stop()
}
