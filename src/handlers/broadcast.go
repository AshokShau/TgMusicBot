/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/core/db"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	broadcastCancelFlag atomic.Bool
)

func cancelBroadcastHandler(m *tg.NewMessage) error {
	broadcastCancelFlag.Store(true)
	_, _ = m.Reply("🚫 Broadcast cancelled.")
	return tg.EndGroup
}

func broadcastHandler(m *tg.NewMessage) error {
	ctx, cancel := db.Ctx()
	defer cancel()

	reply, err := m.GetReplyMessage()
	if err != nil {
		_, _ = m.Reply("❗ Reply to a message to broadcast.\nExample:\n`/broadcast -copy -limit 100 -delay 2s optional preview text`")
		return tg.EndGroup
	}

	args := strings.Fields(m.Args())
	if len(args) == 0 {
		_, _ = m.Reply("Provide flags.\nExample: `/broadcast -copy -limit 50 -delay 1s`")
		return tg.EndGroup
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
				return tg.EndGroup
			}
			limit = n

		case strings.HasPrefix(a, "-delay"):
			val := strings.TrimPrefix(a, "-delay")
			val = strings.TrimSpace(val)
			d, err := time.ParseDuration(val)
			if err != nil {
				_, _ = m.Reply("❗ Invalid delay. Example: `-delay 2s`")
				return tg.EndGroup
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
		return tg.EndGroup
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

	workers := 20
	jobs := make(chan int64, workers)
	wg := sync.WaitGroup{}

	worker := func() {
		for id := range jobs {
			if broadcastCancelFlag.Load() {
				atomic.AddInt32(&failed, 1)
				continue
			}

			for {
				_, errSend := reply.ForwardTo(id, &tg.ForwardOptions{
					Noforwards: copyMode,
				})

				if errSend == nil {
					atomic.AddInt32(&success, 1)
					break
				}

				if wait := tg.GetFloodWait(errSend); wait > 0 {
					gologging.Warn("FloodWait %ds for chatID=%d", wait, id)
					time.Sleep(time.Duration(wait) * time.Second)
					continue
				}

				atomic.AddInt32(&failed, 1)
				gologging.WarnF("[Broadcast] chatID: %d error: %v", id, errSend)
				break
			}

			if delay > 0 {
				time.Sleep(delay)
			}
		}
		wg.Done()
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}

	for _, id := range targets {
		jobs <- id
	}
	close(jobs)

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
	return tg.EndGroup
}
