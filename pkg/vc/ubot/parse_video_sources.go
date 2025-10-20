/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

import (
	"github.com/AshokShau/TgMusicBot/pkg/vc/ntgcalls"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func parseVideoSources(sources []*tg.GroupCallParticipantVideoSourceGroup) []ntgcalls.SsrcGroup {
	var ssrcGroups []ntgcalls.SsrcGroup
	for _, source := range sources {
		var sourceIds []uint32
		for _, sourceId := range source.Sources {
			sourceIds = append(sourceIds, uint32(sourceId))
		}
		ssrcGroups = append(ssrcGroups, ntgcalls.SsrcGroup{
			Semantics: source.Semantics,
			Ssrcs:     sourceIds,
		})
	}
	return ssrcGroups
}
