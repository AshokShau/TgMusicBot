/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package types

import (
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type CallParticipantsCache struct {
	CallParticipants  map[int64]*tg.GroupCallParticipant
	LastMtprotoUpdate time.Time
}
