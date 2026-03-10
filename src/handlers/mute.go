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

// muteHandler handles the /mute command.
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
		_, err := m.ReplyText(c, "⏸ No track currently playing.", nil)
		return err
	}

	if _, err := vc.Calls.Mute(chatID); err != nil {
		_, err = m.ReplyText(c, fmt.Sprintf("❌ An error occurred while muting the playback: %s", err.Error()), nil)
		return err
	}

	user, err := c.GetUser(m.SenderID())
	if err != nil {
		user = &td.User{FirstName: "Unknown"}
	}

	_, err = m.ReplyText(c, fmt.Sprintf("🔇 Playback has been muted by %s.", user.FirstName), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("mute")})
	return err
}

// unmuteHandler handles the /unmute command.
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
		_, err := m.ReplyText(c, "⏸ No track currently playing.", nil)
		return err
	}

	if _, err := vc.Calls.Unmute(chatID); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ An error occurred while unmuting the playback: %s", err.Error()), nil)
		return err
	}

	user, err := c.GetUser(m.SenderID())
	if err != nil {
		user = &td.User{FirstName: "Unknown"}
	}

	_, err = m.ReplyText(c, fmt.Sprintf("🔊 Playback has been unmuted by %s.", user.FirstName), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("unmute")})
	return err
}
