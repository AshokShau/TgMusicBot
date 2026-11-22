/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package downloader

import (
	td "github.com/AshokShau/gotdbot"
	"github.com/AshokShau/gotdbot/logger"
)

var (
	DlBot *td.Client
	log   = logger.New()
)
