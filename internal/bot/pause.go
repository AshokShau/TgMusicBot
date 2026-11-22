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

	td "github.com/AshokShau/gotdbot"
)

func pauseHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "There is no active playback in the video chat.", nil)
		return nil
	}

	if _, err := calls.Calls.Pause(chatID); err != nil {
		c.Logger.Errorf("pause playback in chat %d: %v", chatID, err)
		_, replyErr := m.ReplyText(c, "Unable to pause playback. Please try again.", nil)
		if replyErr != nil {
			c.Logger.Warnf("send pause failure in chat %d: %v", chatID, replyErr)
		}
		return replyErr
	}

	_, err := m.ReplyText(c, fmt.Sprintf("Playback has been paused by %s.", firstName(c, m)), &td.SendMessageOpts{ReplyMarkup: utils.ControlButtons("pause")})
	if err != nil {
		c.Logger.Warnf("send pause confirmation in chat %d: %v", chatID, err)
	}
	return err
}

func resumeHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if chatID > 0 {
		_, _ = m.ReplyText(c, "This command can only be used in a supergroup.", nil)
		return nil
	}

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "There is no active playback in the video chat.", nil)
		return nil
	}

	if _, err := calls.Calls.Resume(chatID); err != nil {
		c.Logger.Errorf("resume playback in chat %d: %v", chatID, err)
		_, replyErr := m.ReplyText(c, "Unable to resume playback. Please try again.", nil)
		if replyErr != nil {
			c.Logger.Warnf("send resume failure in chat %d: %v", chatID, replyErr)
		}
		return replyErr
	}

	_, err := m.ReplyText(c, fmt.Sprintf("Playback has been resumed by %s.", firstName(c, m)), &td.SendMessageOpts{ReplyMarkup: utils.ControlButtons("resume")})
	if err != nil {
		c.Logger.Warnf("send resume confirmation in chat %d: %v", chatID, err)
	}
	return err
}
