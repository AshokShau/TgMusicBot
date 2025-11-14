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
	"strings"
	"sync"
	"time"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"
)

func broadCastHandler(m *tg.NewMessage) error {
	ctx, cancel := db.Ctx()
	defer cancel()
	reply, err := m.GetReplyMessage()
	if err != nil {
		_, _ = m.Reply(
			"❗ <b>Usage:</b> Reply to a message and send:\n" +
				"<code>/broadcast</code>\n" +
				"<code>`/broadcast -copy</code>\n" +
				"<code>/broadcast -nochat</code>\n" +
				"<code>/broadcast -nouser</code>\n\n" +
				"Multiple flags allowed (e.g. <code>-copy -nochat</code>)",
		)
		return tg.EndGroup
	}

	args := strings.Fields(strings.ToLower(m.Args()))
	copyMode := false
	noChats := false
	noUsers := false

	for _, a := range args {
		switch a {
		case "-copy":
			copyMode = true
		case "-nochat", "-nochats":
			noChats = true
		case "-nouser", "-nousers":
			noUsers = true
		}
	}

	chats, err := db.Instance.GetAllChats(ctx)
	if err != nil {
		gologging.Error("GetAllChats: %v", err)
	}

	users, err := db.Instance.GetAllUsers(ctx)
	if err != nil {
		gologging.Error("GetAllUsers: %v", err)
	}

	var targets []int64
	if !noChats {
		targets = append(targets, chats...)
	}

	if !noUsers {
		targets = append(targets, users...)
	}

	if len(targets) == 0 {
		_, _ = m.Reply("⚠️ No valid targets to broadcast.")
		return tg.EndGroup
	}

	sentMsg, _ := m.Reply(fmt.Sprintf("📡 Broadcasting to %d targets...", len(targets)))

	total := len(targets)
	success := 0
	failed := 0

	workers := 20
	jobs := make(chan int64, workers)
	wg := sync.WaitGroup{}

	sendToTarget := func(chatID int64) {
		defer wg.Done()

		for {
			_, errSend := reply.ForwardTo(chatID, &tg.ForwardOptions{
				Noforwards: copyMode,
			})

			if errSend == nil {
				success++
				return
			}

			if wait := tg.GetFloodWait(errSend); wait > 0 {
				gologging.Warn("FloodWait %ds → retry chatID=%d", wait, chatID)
				time.Sleep(time.Duration(wait) * time.Second)
				continue
			}

			failed++
			gologging.WarnF("[Broadcast] chatID=%d error=%v", chatID, errSend)
			return
		}
	}

	for i := 0; i < workers; i++ {
		go func() {
			for id := range jobs {
				wg.Add(1)
				sendToTarget(id)
			}
		}()
	}

	for _, id := range targets {
		jobs <- id
	}
	close(jobs)

	wg.Wait()

	text := fmt.Sprintf(
		"📢 <b>Broadcast Complete</b>\n\n"+
			"👥 Total Targets: %d\n"+
			"✅ Success: %d\n"+
			"❌ Failed: %d\n"+
			"⚙ Mode: %s\n",
		total,
		success,
		failed,
		func() string {
			if copyMode {
				return "Copy"
			}
			return "Forward"
		}(),
	)

	_, _ = sentMsg.Edit(text)
	gologging.Info("[Broadcast completed] total=%d success=%d failed=%d", total, success, failed)
	return tg.EndGroup
}
