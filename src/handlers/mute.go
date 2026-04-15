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

func muteHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}

	m := ctx.EffectiveMessage
	if args := Args(m); args != "" {
		return td.EndGroups
	}

	chatID := m.ChatId
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "There is no active playback in the video chat.", nil)
		return err
	}

	if _, err := vc.Calls.Mute(chatID); err != nil {
		_, err = m.ReplyText(c, fmt.Sprintf("Failed to mute the playback: %s", err.Error()), nil)
		return err
	}

	_, err := m.ReplyText(c, fmt.Sprintf("Playback has been muted by %s.", firstName(c, m)), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("mute")})
	return err
}

func unmuteHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}

	m := ctx.EffectiveMessage
	if args := Args(m); args != "" {
		return td.EndGroups
	}

	chatID := m.ChatId
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "There is no active playback in the video chat.", nil)
		return err
	}

	if _, err := vc.Calls.Unmute(chatID); err != nil {
		_, err = m.ReplyText(c, fmt.Sprintf("Failed to unmute the playback: %s", err.Error()), nil)
		return err
	}

	_, err := m.ReplyText(c, fmt.Sprintf("Playback has been unmuted by %s.", firstName(c, m)), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("unmute")})
	return err
}
