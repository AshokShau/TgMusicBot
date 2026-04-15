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
	"strconv"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// speedHandler handles the /speed command.
func speedHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}
	chatID := ctx.EffectiveChatId
	m := ctx.EffectiveMessage

	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "The bot is not streaming in the video chat.", nil)
		return err
	}

	if playingSong := cache.ChatCache.GetPlayingTrack(chatID); playingSong == nil {
		_, err := m.ReplyText(c, "The bot is not streaming in the video chat.", nil)
		return err
	}

	args := Args(m)
	if args == "" {
		_, _ = m.ReplyText(c, "<b>Change Playback Speed</b>\n\n<b>Usage:</b> <code>/speed [value]</code>\n\nThe speed can be set between <code>0.5</code> and <code>4.0</code>.", replyOpts)
		return nil
	}

	speed, err := strconv.ParseFloat(args, 64)
	if err != nil {
		_, _ = m.ReplyText(c, "Invalid speed value. Please provide a number between 0.5 and 4.0.", nil)
		return nil
	}

	if err = vc.Calls.ChangeSpeed(chatID, speed); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("An error occurred while changing the speed: %s", err.Error()), replyOpts)
		return nil
	}

	_, _ = m.ReplyText(c, fmt.Sprintf("Playback speed has been set to <code>%.2fx</code>.", speed), replyOpts)
	return nil
}
