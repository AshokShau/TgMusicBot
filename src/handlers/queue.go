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
	"math"
	"strconv"
	"strings"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// queueHandler displays the current playback queue with detailed information.
func queueHandler(c *td.Client, ctx *td.Context) error {
	if !adminMode(c, ctx) {
		return td.EndGroups
	}

	m := ctx.EffectiveMessage
	chatID := ctx.EffectiveChatId

	chat, err := c.GetChat(chatID)
	if err != nil {
		_, _ = m.ReplyText(c, "Error fetching chat information.", nil)
		return nil
	}

	queue := cache.ChatCache.GetQueue(chatID)
	if len(queue) == 0 {
		_, _ = m.ReplyText(c, "The queue is empty.", nil)
		return nil
	}

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "The bot is not streaming in the video chat.", nil)
		return nil
	}

	current := queue[0]
	playedTime, _ := vc.Calls.PlayedTime(chatID)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("<b>Queue for %s</b>\n\n", chat.Title))

	b.WriteString("<b>Now Playing:</b>\n")
	b.WriteString(fmt.Sprintf("• <b>Title:</b> <code>%s</code>\n", truncate(current.Name, 45)))
	b.WriteString(fmt.Sprintf("• <b>By:</b> %s\n", current.User))
	b.WriteString(fmt.Sprintf("• <b>Duration:</b> %s min\n", utils.SecToMin(current.Duration)))
	b.WriteString("• <b>Loop:</b> ")
	if current.Loop > 0 {
		b.WriteString("On\n")
	} else {
		b.WriteString("Off\n")
	}
	b.WriteString("• <b>Progress:</b> ")
	if playedTime > 0 && playedTime < math.MaxInt {
		b.WriteString(utils.SecToMin(int(playedTime)))
	} else {
		b.WriteString("0:00")
	}
	b.WriteString(" min\n")

	if len(queue) > 1 {
		b.WriteString(fmt.Sprintf("\n<b>Next Up (%d):</b>\n", len(queue)-1))

		for i, song := range queue[1:] {
			if i >= 14 {
				break
			}
			b.WriteString(strconv.Itoa(i + 1))
			b.WriteString(". <code>")
			b.WriteString(truncate(song.Name, 45))
			b.WriteString("</code> | ")
			b.WriteString(utils.SecToMin(song.Duration))
			b.WriteString(" min\n")
		}

		if len(queue) > 15 {
			b.WriteString(fmt.Sprintf("...and %d more tracks\n", len(queue)-15))
		}
	}

	b.WriteString(fmt.Sprintf("\n<b>Total:</b> %d tracks", len(queue)))

	text := b.String()
	if len(text) > 4096 {
		var sb strings.Builder
		progress := "0:00"
		if playedTime > 0 && playedTime < math.MaxInt {
			progress = utils.SecToMin(int(playedTime))
		}
		sb.WriteString(fmt.Sprintf(
			"<b>Queue for %s</b>\n\n<b>Now Playing:</b>\n• <code>%s</code>\n• %s/%s min\n\n<b>Total:</b> %d tracks",
			chat.Title,
			truncate(current.Name, 45),
			progress,
			utils.SecToMin(current.Duration),
			len(queue),
		))
		text = sb.String()
	}

	_, err = m.ReplyText(c, text, replyOpts)
	return err
}
