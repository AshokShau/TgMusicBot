/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/core/db"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	td "github.com/AshokShau/gotdbot"
)

var (
	broadcastCancelFlag atomic.Bool
	broadcastInProgress atomic.Bool
)

func cancelBroadcastHandler(c *td.Client, ctx *td.Context) error {
	if !isDev(ctx) {
		return td.EndGroups
	}
	m := ctx.EffectiveMessage
	broadcastCancelFlag.Store(true)
	_, _ = m.ReplyText(c, "🚫 Broadcast cancelled.", nil)
	return td.EndGroups
}

func broadcastHandler(c *td.Client, ctx *td.Context) error {
	if !isDev(ctx) {
		return td.EndGroups
	}

	m := ctx.EffectiveMessage
	if broadcastInProgress.Load() {
		_, _ = m.ReplyText(c, "❗ A broadcast is already in progress. Please wait for it to complete or cancel it with /cancelbroadcast", nil)
		return td.EndGroups
	}

	broadcastInProgress.Store(true)
	defer broadcastInProgress.Store(false)

	ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reply, err := m.GetRepliedMessage(c)
	if err != nil {
		_, _ = m.ReplyText(c, "❗ Reply to a message to broadcast.\nExample:\n<code> /broadcast -copy -limit 100 -delay 2s</code>", &td.SendTextMessageOpts{ParseMode: "HTMl"})
		return td.EndGroups
	}

	args := strings.Fields(Args(m))
	copyMode := false
	noChats := false
	noUsers := false
	limit := 0
	delay := time.Duration(0)

	for _, a := range args {
		switch {
		case a == "-copy":
			copyMode = true
		case a == "-nochat" || a == "-nochats":
			noChats = true
		case a == "-nouser" || a == "-nousers":
			noUsers = true

		case strings.HasPrefix(a, "-limit"):
			val := strings.TrimPrefix(a, "-limit")
			val = strings.TrimSpace(val)
			n, err := strconv.Atoi(val)
			if err != nil || n <= 0 {
				_, _ = m.ReplyText(c, "❗ Invalid limit value. Example: <code>-limit 100</code>", &td.SendTextMessageOpts{ParseMode: "HTMl"})
				return td.EndGroups
			}
			limit = n

		case strings.HasPrefix(a, "-delay"):
			val := strings.TrimPrefix(a, "-delay")
			val = strings.TrimSpace(val)
			d, err := time.ParseDuration(val)
			if err != nil {
				_, _ = m.ReplyText(c, "❗ Invalid delay. Example: <code>-delay 2s</code>", &td.SendTextMessageOpts{ParseMode: "HTMl"})
				return td.EndGroups
			}
			delay = d
		}
	}

	broadcastCancelFlag.Store(false)
	chats, _ := db.Instance.GetAllChats(ctx2)
	users, _ := db.Instance.GetAllUsers(ctx2)

	var targets []int64
	if !noChats {
		targets = append(targets, chats...)
	}
	if !noUsers {
		targets = append(targets, users...)
	}

	if len(targets) == 0 {
		_, _ = m.ReplyText(c, "❗ No targets found.", nil)
		return td.EndGroups
	}

	if limit > 0 && limit < len(targets) {
		targets = targets[:limit]
	}

	sentMsg, _ := m.ReplyText(c,
		fmt.Sprintf(
			"🚀 <b>Broadcast Started</b>\nTargets: %d\nMode: %s\nDelay: %v\n\nSend <code>/cancelbroadcast</code> to stop.",
			len(targets),
			map[bool]string{true: "Copy", false: "Forward"}[copyMode],
			delay,
		), &td.SendTextMessageOpts{ParseMode: "HTML"})

	var success int32
	var failed int32

	type QueueItem struct {
		ID         int64
		RetryCount int
	}

	queue := make([]QueueItem, len(targets))
	for i, id := range targets {
		queue[i] = QueueItem{ID: id, RetryCount: 0}
	}
	var queueMutex sync.Mutex

	interval := time.Second / 25
	if delay > 0 {
		interval = delay
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	workers := 20
	wg := sync.WaitGroup{}

	worker := func() {
		for {
			queueMutex.Lock()
			if len(queue) == 0 {
				queueMutex.Unlock()
				break
			}
			item := queue[0]
			queue = queue[1:]
			queueMutex.Unlock()

			if broadcastCancelFlag.Load() {
				atomic.AddInt32(&failed, 1)
				continue
			}

			<-ticker.C

			var errSend error
			if copyMode {
				_, errSend = reply.Copy(c, item.ID, &td.SendCopyOpts{
					ReplyMarkup: reply.ReplyMarkup,
				})
			} else {
				_, errSend = reply.Forward(c, item.ID, &td.ForwardMessageOpts{})
			}

			if errSend == nil {
				atomic.AddInt32(&success, 1)
				continue
			}

			/*
				if wait := td.GetFloodWait(errSend); wait > 0 {
					c.Logger.Warn("FloodWait s for chatID=", "arg1", wait, "id", item.ID)

					if item.RetryCount < 2 {
						item.RetryCount++
						queueMutex.Lock()
						queue = append(queue, item)
						queueMutex.Unlock()

						time.Sleep(time.Duration(wait) * time.Second)
						continue
					} else {

					}
				}
			*/

			c.Logger.Warn("[Broadcast] max retries reached for chatID", "id", item.ID)
			atomic.AddInt32(&failed, 1)
			c.Logger.Warn("[Broadcast] chatID:  error", "id", item.ID, "error", errSend)
		}
		wg.Done()
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}

	wg.Wait()

	total := len(targets)
	result := fmt.Sprintf(
		"📢 <b>Broadcast Complete</b>\n\n"+
			"👥 Total: %d\n"+
			"✅ Success: %d\n"+
			"❌ Failed: %d\n"+
			"⚙ Mode: %s\n"+
			"⏱ Delay: %v\n"+
			"🛑 Cancelled: %v\n",
		total,
		success,
		failed,
		map[bool]string{true: "Copy", false: "Forward"}[copyMode],
		delay,
		broadcastCancelFlag.Load(),
	)

	_, _ = sentMsg.EditText(c, result, &td.EditTextMessageOpts{ParseMode: "HTML"})
	broadcastInProgress.Store(false)
	return td.EndGroups
}
