/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/utils"
	"fmt"
	"strconv"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// seekHandler handles the /seek command.
func seekHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}
	chatID := ctx.EffectiveChatId
	m := ctx.EffectiveMessage

	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "⏸ No track currently playing.", nil)
		return err
	}

	playingSong := cache.ChatCache.GetPlayingTrack(chatID)
	if playingSong == nil {
		_, err := m.ReplyText(c, "⏸ No track currently playing.", nil)
		return err
	}

	args := Args(m)
	if args == "" {
		_, _ = m.ReplyText(c, "<b>❌ Seek Track</b>\n\n<b>Usage:</b> <code>/seek [seconds]</code>", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return nil
	}

	seekTime, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.ReplyText(c, "❌ Invalid seek time provided. Please use a valid number of seconds.", nil)
		return nil
	}

	if seekTime < 0 || seekTime < 20 {
		_, _ = m.ReplyText(c, "⚠️ The minimum seek time is 20 seconds.", nil)
		return nil
	}

	currDur, err := vc.Calls.PlayedTime(chatID)
	if err != nil {
		_, _ = m.ReplyText(c, "❌ An error occurred while fetching the current track duration.", nil)
		return nil
	}

	toSeek := int(currDur) + seekTime
	if toSeek >= playingSong.Duration {
		_, _ = m.ReplyText(c, fmt.Sprintf("⚠️ You cannot seek beyond the track's duration. The maximum seek time is %s.", utils.SecToMin(playingSong.Duration)), nil)
		return nil
	}

	if err = vc.Calls.SeekStream(chatID, playingSong.FilePath, toSeek, playingSong.Duration, playingSong.IsVideo); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ An error occurred while seeking the track: %s", err.Error()), nil)
		return nil
	}

	_, _ = m.ReplyText(c, fmt.Sprintf("✅ The track has been seeked to %s.", utils.SecToMin(toSeek)), nil)
	return nil
}
