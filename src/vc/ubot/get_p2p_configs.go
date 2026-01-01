/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

import (
	"ashokshau/tgmusic/src/vc/ubot/types"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) getP2PConfigs(GAorB []byte) (*types.P2PConfig, error) {
	dhConfigRaw, err := ctx.App.MessagesGetDhConfig(0, 256)
	if err != nil {
		return nil, err
	}

	dhConfig := dhConfigRaw.(*tg.MessagesDhConfigObj)
	return &types.P2PConfig{
		DhConfig:   dhConfig,
		IsOutgoing: GAorB == nil,
		GAorB:      GAorB,
		WaitData:   make(chan error),
	}, nil
}
