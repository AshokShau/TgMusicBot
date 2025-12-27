/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package pkg

import (
	"ashokshau/tgmusic/src/config"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/handlers"
	"ashokshau/tgmusic/src/vc"
	"context"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func Init(client *tg.Client) error {
	if err := db.InitDatabase(context.Background()); err != nil {
		return err
	}
	// Then start the voice call clients
	for _, session := range config.Conf.SessionStrings {
		_, err := vc.Calls.StartClient(config.Conf.ApiId, config.Conf.ApiHash, session)
		if err != nil {
			return err
		}
	}

	// Leave all chats on startup (except logger group)
	vc.Calls.LeaveAllOnStartup()

	// Start auto leave service
	vc.Calls.StartAutoLeaveService()

	// Register handlers and load modules
	vc.Calls.RegisterHandlers(client)
	handlers.LoadModules(client)

	return nil
}
