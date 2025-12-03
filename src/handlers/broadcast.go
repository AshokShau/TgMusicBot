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

	ctx, cancel := db.Ctx()
	defer cancel()

	reply, err := m.GetReplyMessage()
	if err != nil {
		_, _ = m.Reply("❗ Reply to a message to broadcast.\n\nFlags:\n<code>-copy</code> - Send as copy (hide forward tag)\n<code>-nochats</code> - Skip groups\n<code>-nousers</code> - Skip users\n<code>-nochannels</code> - Skip channels\n\nExample: <code>/broadcast -copy</code>")
		return tg.ErrEndGroup
	}

	args := strings.Fields(m.Args())

	copyMode := false
	noChats := false
	noUsers := false
	noChannels := false

	for _, a := range args {
		switch {
		case a == "-copy":
			copyMode = true
		case a == "-nochat" || a == "-nochats":
			noChats = true
		case a == "-nouser" || a == "-nousers":
			noUsers = true
		case a == "-nochannel" || a == "-nochannels":
			noChannels = true
		}
	}

	broadcastCancelFlag.Store(false)
	chats, _ := db.Instance.GetAllChats(ctx)
	users, _ := db.Instance.GetAllUsers(ctx)

	var targets []int64
	var groups []int64
	var channels []int64

	// Separate groups and channels from chats
	// Channels have IDs starting with -100 and are typically < -1000000000000
	// Groups also start with -100 but we need to check via API or assume all negative IDs in chats are groups/channels
	for _, chatID := range chats {
		if chatID < -1000000000000 {
			// This is likely a channel (supergroup/channel format)
			channels = append(channels, chatID)
		} else if chatID < 0 {
			// This is a group
			groups = append(groups, chatID)
		}
	}

	if !noChats {
		targets = append(targets, groups...)
	}
	if !noChannels {
		targets = append(targets, channels...)
	}
	if !noUsers {
		targets = append(targets, users...)
	}

	if len(targets) == 0 {
		_, _ = m.Reply("❗ No targets found.")
		return tg.ErrEndGroup
	}

	sentMsg, _ := m.Reply(fmt.Sprintf(
		"🚀 <b>Broadcast Started</b>\n\n👥 Users: %d\n💬 Groups: %d\n📢 Channels: %d\n📊 Total: %d\n⚙ Mode: %s\n\nSend <code>/cancelbroadcast</code> to stop.",
		len(users),
		len(groups),
		len(channels),
		len(targets),
		map[bool]string{true: "Copy", false: "Forward"}[copyMode],
	))

	var success int32
	var failed int32
	var removed int32

	workers := 20
	jobs := make(chan int64, workers)
	wg := sync.WaitGroup{}

	// shouldRemoveFromDB checks if the error indicates the user/chat should be removed
	shouldRemoveFromDB := func(err error) bool {
		if err == nil {
			return false
		}
		errStr := strings.ToLower(err.Error())
		// User blocked the bot
		if strings.Contains(errStr, "user_is_blocked") {
			return true
		}
		// User deleted their account
		if strings.Contains(errStr, "user_is_deleted") {
			return true
		}
		// User deactivated
		if strings.Contains(errStr, "user_deactivated") {
			return true
		}
		// Invalid peer ID (user/chat doesn't exist)
		if strings.Contains(errStr, "peer_id_invalid") {
			return true
		}
		// Chat not found / no channel with id
		if strings.Contains(errStr, "no channel with id") {
			return true
		}
		// Chat was deleted
		if strings.Contains(errStr, "chat_write_forbidden") {
			return true
		}
		// Bot was kicked from the chat
		if strings.Contains(errStr, "bot_kicked") {
			return true
		}
		// Chat not found
		if strings.Contains(errStr, "chat_not_found") {
			return true
		}
		// Input user deactivated
		if strings.Contains(errStr, "input_user_deactivated") {
			return true
		}
		return false
	}

	// removeFromDB removes the ID from the appropriate collection
	removeFromDB := func(id int64) {
		dbCtx, dbCancel := db.Ctx()
		defer dbCancel()

		if id > 0 {
			// It's a user
			if err := db.Instance.RemoveUser(dbCtx, id); err == nil {
				atomic.AddInt32(&removed, 1)
				logger.Info("[Broadcast] Removed user %d from database", id)
			}
		} else {
			// It's a chat/channel
			if err := db.Instance.RemoveChat(dbCtx, id); err == nil {
				atomic.AddInt32(&removed, 1)
				logger.Info("[Broadcast] Removed chat %d from database", id)
			}
		}
	}

	worker := func() {
		for id := range jobs {
			if broadcastCancelFlag.Load() {
				atomic.AddInt32(&failed, 1)
				continue
			}

			for {
				_, errSend := reply.ForwardTo(id, &tg.ForwardOptions{
					HideAuthor: copyMode,
				})

				if errSend == nil {
					atomic.AddInt32(&success, 1)
					break
				}

				if wait := tg.GetFloodWait(errSend); wait > 0 {
					logger.Warn("FloodWait %ds for chatID=%d", wait, id)
					time.Sleep(time.Duration(wait) * time.Second)
					continue
				}

				atomic.AddInt32(&failed, 1)
				logger.Warn("[Broadcast] chatID: %d error: %v", id, errSend)

				// Check if we should remove this ID from database
				if shouldRemoveFromDB(errSend) {
					removeFromDB(id)
				}
				break
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
			"🗑 Removed: %d\n"+
			"⚙ Mode: %s\n"+
			"🛑 Cancelled: %v\n",
		total,
		success,
		failed,
		removed,
		map[bool]string{true: "Copy", false: "Forward"}[copyMode],
		broadcastCancelFlag.Load(),
	)

	_, _ = sentMsg.Edit(result)
	broadcastInProgress.Store(false)
	return tg.ErrEndGroup
}
