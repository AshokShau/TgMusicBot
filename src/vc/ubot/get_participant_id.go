/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

import tg "github.com/amarnathcjd/gogram/telegram"

func getParticipantId(peer tg.Peer) int64 {
	var participantId int64
	switch chatObj := peer.(type) {
	case *tg.PeerUser:
		participantId = chatObj.UserID
	case *tg.PeerChannel:
		participantId = chatObj.ChannelID
	case *tg.PeerChat:
		participantId = chatObj.ChatID
	}
	return participantId
}
