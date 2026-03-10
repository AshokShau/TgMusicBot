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

// pauseHandler handles the /pause command.
func pauseHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}
	m := ctx.EffectiveMessage
	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "⏸ No track currently playing.", nil)
		return nil
	}

	if _, err := vc.Calls.Pause(chatID); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ An error occurred while pausing the playback: %s", err.Error()), nil)
		return nil
	}

	user, err := c.GetUser(m.SenderID())
	if err != nil {
		user = &td.User{FirstName: "Unknown"}
	}

	_, err = m.ReplyText(c, fmt.Sprintf("⏸️ Playback has been paused by %s.", user.FirstName), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("pause")})
	return err
}

// resumeHandler handles the /resume command.
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
		_, _ = m.ReplyText(c, "⏸ No track currently playing.", nil)
		return nil
	}

	if _, err := vc.Calls.Resume(chatID); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ An error occurred while resuming the playback: %s", err.Error()), nil)
		return nil
	}

	user, err := c.GetUser(m.SenderID())
	if err != nil {
		user = &td.User{FirstName: "Unknown", LastName: ""}
	}

	_, err = m.ReplyText(c, fmt.Sprintf("▶️ Playback has been resumed by %s.", user.FirstName), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("resume")})
	return err
}
