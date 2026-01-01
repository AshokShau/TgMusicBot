/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package types

import tg "github.com/amarnathcjd/gogram/telegram"

type P2PConfig struct {
	DhConfig       *tg.MessagesDhConfigObj
	PhoneCall      *tg.PhoneCallObj
	IsOutgoing     bool
	KeyFingerprint int64
	GAorB          []byte
	WaitData       chan error
}
