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
		_, err := m.ReplyText(c, "The bot is not streaming in the video chat.", nil)
		return err
	}

	playingSong := cache.ChatCache.GetPlayingTrack(chatID)
	if playingSong == nil {
		_, err := m.ReplyText(c, "The bot is not streaming in the video chat.", nil)
		return err
	}

	args := Args(m)
	if args == "" {
		_, _ = m.ReplyText(c, "<b>Usage:</b> /seek duration\n<b>Example:</b> <code>/seek 15</code>", replyOpts)
		return nil
	}

	seekTime, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.ReplyText(c, "Invalid seek time provided. Please use a valid number of seconds.", nil)
		return nil
	}

	if seekTime < 10 {
		_, _ = m.ReplyText(c, "Minimum seek time is 10 seconds.", nil)
		return nil
	}

	currDur, err := vc.Calls.PlayedTime(chatID)
	if err != nil {
		_, _ = m.ReplyText(c, "Failed to fetch the duration of the ongoing stream.", nil)
		return nil
	}

	toSeek := int(currDur) + seekTime
	if toSeek >= playingSong.Duration {
		_, _ = m.ReplyText(c, fmt.Sprintf("You cannot seek beyond the track duration. Maximum allowed is %s.", utils.SecToMin(playingSong.Duration)), nil)
		return nil
	}

	if err = vc.Calls.SeekStream(
		chatID,
		playingSong.FilePath,
		toSeek,
		playingSong.Duration,
		playingSong.IsVideo,
	); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("An error occurred while seeking the track: %s", err.Error()), replyOpts)
		return nil
	}

	_, _ = m.ReplyText(c, fmt.Sprintf("<b>Stream skipped %s and started from %s seconds by</b> %s", utils.SecToMin(seekTime), utils.SecToMin(toSeek), firstName(c, m)), replyOpts)
	return nil
}
