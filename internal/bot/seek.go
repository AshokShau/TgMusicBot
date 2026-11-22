/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tgmusic/internal/cache"
	"ashokshau/tgmusic/internal/calls"
	"ashokshau/tgmusic/internal/utils"
	"fmt"
	"strconv"

	td "github.com/AshokShau/gotdbot"
)

func seekHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}
	chatID := m.ChatId

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

	currDur, err := calls.Calls.PlayedTime(chatID)
	if err != nil {
		c.Logger.Warnf("read playback time in chat %d: %v", chatID, err)
		_, replyErr := m.ReplyText(c, "Unable to read playback time. Please try again.", nil)
		if replyErr != nil {
			c.Logger.Warnf("send playback time failure in chat %d: %v", chatID, replyErr)
		}
		return replyErr
	}

	toSeek := int32(currDur) + int32(seekTime)
	if toSeek >= playingSong.Duration {
		_, _ = m.ReplyText(c, fmt.Sprintf("You cannot seek beyond the track duration. Maximum allowed is %s.", utils.SecToMin(playingSong.Duration)), nil)
		return nil
	}

	if err = calls.Calls.SeekStream(c, chatID, seekTime); err != nil {
		c.Logger.Errorf("seek playback in chat %d: %v", chatID, err)
		_, replyErr := m.ReplyText(c, "Unable to seek playback. Please try again.", nil)
		return replyErr
	}

	_, replyErr := m.ReplyText(c, fmt.Sprintf("<b>Stream skipped %s and started from %s seconds by</b> %s", utils.SecToMin(int32(seekTime)), utils.SecToMin(toSeek), firstName(c, m)), replyOpts)
	return replyErr
}
