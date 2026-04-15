/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"fmt"

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/vc"

	td "github.com/AshokShau/gotdbot"
)

func pauseHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}

	m := ctx.EffectiveMessage
	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "There is no active playback in the video chat.", nil)
		return nil
	}

	if _, err := vc.Calls.Pause(chatID); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("Failed to pause the playback: %s", err.Error()), nil)
		return nil
	}

	_, err := m.ReplyText(c, fmt.Sprintf("Playback has been paused by %s.", firstName(c, m)), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("pause")})
	return err
}

func resumeHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}
	m := ctx.EffectiveMessage
	chatID := m.ChatId

	if chatID > 0 {
		_, _ = m.ReplyText(c, "This command can only be used in a supergroup.", nil)
		return nil
	}

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "There is no active playback in the video chat.", nil)
		return nil
	}

	if _, err := vc.Calls.Resume(chatID); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("Failed to resume the playback: %s", err.Error()), nil)
		return nil
	}

	_, err := m.ReplyText(c, fmt.Sprintf("Playback has been resumed by %s.", firstName(c, m)), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("resume")})
	return err
}
