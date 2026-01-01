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

	"github.com/amarnathcjd/gogram/telegram"
)

// seekHandler handles the /seek command.
func seekHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.Reply("⏸ No track currently playing.")
		return err
	}

	playingSong := cache.ChatCache.GetPlayingTrack(chatID)
	if playingSong == nil {
		_, err := m.Reply("⏸ No track currently playing.")
		return err
	}

	args := m.Args()
	if args == "" {
		_, _ = m.Reply("<b>❌ Seek Track</b>\n\n<b>Usage:</b> <code>/seek [seconds]</code>")
		return nil
	}

	seekTime, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.Reply("❌ Invalid seek time provided. Please use a valid number of seconds.")
		return nil
	}

	if seekTime < 0 || seekTime < 20 {
		_, _ = m.Reply("⚠️ The minimum seek time is 20 seconds.")
		return nil
	}

	currDur, err := vc.Calls.PlayedTime(chatID)
	if err != nil {
		_, _ = m.Reply("❌ An error occurred while fetching the current track duration.")
		return nil
	}

	toSeek := int(currDur) + seekTime
	if toSeek >= playingSong.Duration {
		_, _ = m.Reply(fmt.Sprintf("⚠️ You cannot seek beyond the track's duration. The maximum seek time is %s.", utils.SecToMin(playingSong.Duration)))
		return nil
	}

	if err = vc.Calls.SeekStream(chatID, playingSong.FilePath, toSeek, playingSong.Duration, playingSong.IsVideo); err != nil {
		_, _ = m.Reply(fmt.Sprintf("❌ An error occurred while seeking the track: %s", err.Error()))
		return nil
	}

	_, _ = m.Reply(fmt.Sprintf("✅ The track has been seeked to %s.", utils.SecToMin(toSeek)))
	return nil
}
