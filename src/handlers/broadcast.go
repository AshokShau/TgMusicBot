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

	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	broadcastCancelFlag atomic.Bool
	broadcastInProgress atomic.Bool
)

func cancelBroadcastHandler(m *tg.NewMessage) error {
	broadcastCancelFlag.Store(true)
	_, _ = m.Reply("🚫 Broadcast cancelled.")
	return tg.ErrEndGroup
}

func broadcastHandler(m *tg.NewMessage) error {
	if broadcastInProgress.Load() {
		_, _ = m.Reply("❗ A broadcast is already in progress. Please wait for it to complete or cancel it with /cancelbroadcast")
		return tg.ErrEndGroup
	}

	broadcastInProgress.Store(true)
	defer broadcastInProgress.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reply, err := m.GetReplyMessage()
	if err != nil {
		_, _ = m.Reply("❗ Reply to a message to broadcast.\nExample:\n`/broadcast -copy -limit 100 -delay 2s optional preview text`")
		return tg.ErrEndGroup
	}

	args := strings.Fields(m.Args())
	if len(args) == 0 {
		_, _ = m.Reply("Provide flags.\nExample: `/broadcast -copy -limit 50 -delay 1s`")
		return tg.ErrEndGroup
	}

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
				_, _ = m.Reply("❗ Invalid limit value. Example: `-limit 100`")
				return tg.ErrEndGroup
			}
			limit = n

		case strings.HasPrefix(a, "-delay"):
			val := strings.TrimPrefix(a, "-delay")
			val = strings.TrimSpace(val)
			d, err := time.ParseDuration(val)
			if err != nil {
				_, _ = m.Reply("❗ Invalid delay. Example: `-delay 2s`")
				return tg.ErrEndGroup
			}
			delay = d
		}
	}

	broadcastCancelFlag.Store(false)
	chats, _ := db.Instance.GetAllChats(ctx)
	users, _ := db.Instance.GetAllUsers(ctx)

	var targets []int64
	if !noChats {
		targets = append(targets, chats...)
	}
	if !noUsers {
		targets = append(targets, users...)
	}

	if len(targets) == 0 {
		_, _ = m.Reply("❗ No targets found.")
		return tg.ErrEndGroup
	}

	if limit > 0 && limit < len(targets) {
		targets = targets[:limit]
	}

	sentMsg, _ := m.Reply(fmt.Sprintf(
		"🚀 <b>Broadcast Started</b>\nTargets: %d\nMode: %s\nDelay: %v\n\nSend <code>/cancelbroadcast</code> to stop.",
		len(targets),
		map[bool]string{true: "Copy", false: "Forward"}[copyMode],
		delay,
	))

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
				_, errSend = m.Client.SendMessage(item.ID, reply)
			} else {
				_, errSend = reply.ForwardTo(item.ID, &tg.ForwardOptions{HideAuthor: copyMode})
			}

			if errSend == nil {
				atomic.AddInt32(&success, 1)
				continue
			}

			if wait := tg.GetFloodWait(errSend); wait > 0 {
				logger.Warn("FloodWait %ds for chatID=%d", wait, item.ID)

				if item.RetryCount < 2 {
					item.RetryCount++
					queueMutex.Lock()
					queue = append(queue, item)
					queueMutex.Unlock()

					time.Sleep(time.Duration(wait) * time.Second)
					continue
				} else {
					logger.Warn("[Broadcast] max retries reached for chatID: %d", item.ID)
				}
			}

			atomic.AddInt32(&failed, 1)
			logger.Warn("[Broadcast] chatID: %d error: %v", item.ID, errSend)
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

	_, _ = sentMsg.Edit(result)
	broadcastInProgress.Store(false)
	return tg.ErrEndGroup
}
