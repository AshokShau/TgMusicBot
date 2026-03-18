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
	"slices"
)

func (ctx *Context) updateSources(chatId int64) error {
	participants, err := ctx.GetParticipants(chatId)
	if err != nil {
		return err
	}

	chatMutex := ctx.getChatMutex(chatId)
	chatMutex.Lock()
	defer chatMutex.Unlock()

	ctx.stateMutex.Lock()
	if ctx.callSources[chatId] == nil {
		ctx.callSources[chatId] = &types.CallSources{
			CameraSources: make(map[int64]string),
			ScreenSources: make(map[int64]string),
		}
	}
	ctx.stateMutex.Unlock()

	for _, participant := range participants {
		participantId := getParticipantId(participant.Peer)

		ctx.stateMutex.Lock()
		if participant.Video != nil && ctx.callSources[chatId].CameraSources[participantId] == "" {
			endpoint := participant.Video.Endpoint
			ctx.callSources[chatId].CameraSources[participantId] = endpoint
			ctx.stateMutex.Unlock()
			_, err = ctx.binding.AddIncomingVideo(chatId, endpoint, parseVideoSources(participant.Video.SourceGroups))
			if err != nil {
				return err
			}
			ctx.stateMutex.Lock()
		}

		if participant.Presentation != nil && ctx.callSources[chatId].ScreenSources[participantId] == "" {
			endpoint := participant.Presentation.Endpoint
			ctx.callSources[chatId].ScreenSources[participantId] = endpoint
			ctx.stateMutex.Unlock()
			_, err = ctx.binding.AddIncomingVideo(chatId, endpoint, parseVideoSources(participant.Presentation.SourceGroups))
			if err != nil {
				return err
			}
			ctx.stateMutex.Lock()
		}
		ctx.stateMutex.Unlock()

		if participantId == ctx.self.ID && !participant.CanSelfUnmute {
			ctx.stateMutex.Lock()
			if !slices.Contains(ctx.mutedByAdmin, chatId) {
				ctx.mutedByAdmin = append(ctx.mutedByAdmin, chatId)
			}
			ctx.stateMutex.Unlock()
		}
	}
	return nil
}
