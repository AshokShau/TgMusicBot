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
	"maps"
	"slices"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) GetParticipants(chatId int64) ([]*tg.GroupCallParticipant, error) {
	ctx.participantsMutex.Lock()
	defer ctx.participantsMutex.Unlock()
	if ctx.callParticipants[chatId] == nil {
		ctx.callParticipants[chatId] = &types.CallParticipantsCache{
			CallParticipants: make(map[int64]*tg.GroupCallParticipant),
		}
	}
	if time.Since(ctx.callParticipants[chatId].LastMtprotoUpdate) > time.Minute {
		groupCall, err := ctx.getInputGroupCall(chatId)
		if err != nil {
			return nil, err
		}

		ctx.callParticipants[chatId].CallParticipants = make(map[int64]*tg.GroupCallParticipant)
		var nextOffset string
		for {
			res, err := ctx.App.PhoneGetGroupParticipants(
				groupCall,
				[]tg.InputPeer{},
				[]int32{},
				nextOffset,
				0,
			)
			if err != nil {
				return nil, err
			}
			for _, participant := range res.Participants {
				ctx.callParticipants[chatId].CallParticipants[getParticipantId(participant.Peer)] = participant
			}
			if res.NextOffset == "" {
				break
			}
			nextOffset = res.NextOffset
		}
		ctx.callParticipants[chatId].LastMtprotoUpdate = time.Now()
	}
	return slices.Collect(maps.Values(ctx.callParticipants[chatId].CallParticipants)), nil
}
