package cleanup

import (
	"ashokshau/tgmusic/internal/cache"
	"ashokshau/tgmusic/internal/db"
	"ashokshau/tgmusic/internal/messaging"
	"context"
	"strconv"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
)

type sent struct {
	chatID, oldID, id int64
	at                time.Time
}

func Start(c *td.Client) (*Scheduler, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	s := New(func(chatID int64, ids []int64) error {
		return c.DeleteMessages(chatID, ids, &td.DeleteMessagesOpts{Revoke: true})
	}, func(chatID int64) bool {
		settings, err := db.Instance.GetCleanSettings(chatID)
		// A storage outage must not throw away pending jobs.
		return err != nil || settings.AutoDelete
	})

	events := make(chan sent, 8192)
	groupTypes := cache.NewCache[bool](10 * time.Minute)
	enqueue := func(m *td.Message, oldID int64) {
		event, valid := sentEvent(m, oldID)
		if !valid {
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case events <- event:
		default:
			c.Logger.Warn("auto-delete update queue at capacity", "chat_id", m.ChatId)
		}
	}

	c.AddUpdateMessageSendSucceededHandlerGroup(func(_ *td.Client, u *td.UpdateMessageSendSucceeded) error {
		enqueue(u.Message, u.OldMessageId)
		return td.ContinueGroups
	}, nil, -101)

	c.AddUpdateNewMessageHandlerGroup(func(_ *td.Client, u *td.UpdateNewMessage) error { enqueue(u.Message, 0); return td.ContinueGroups }, nil, -2)

	c.AddUpdateDeleteMessagesHandlerGroup(func(_ *td.Client, u *td.UpdateDeleteMessages) error {
		if ctx.Err() == nil {
			s.TelegramDeleted(u)
		}
		return td.ContinueGroups
	}, nil, -2)

	c.AddUpdateChatMemberHandlerGroup(func(_ *td.Client, u *td.UpdateChatMember) error {
		if ctx.Err() != nil {
			return td.ContinueGroups
		}

		cache.UpdateAdminCache(u.ChatId, u.NewChatMember)
		if u.NewChatMember == nil || u.ChatId >= 0 {
			return td.ContinueGroups
		}

		s.TelegramMember(c.Me.Id, u)
		return td.ContinueGroups
	}, nil, -2)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); s.Run(ctx) }()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-events:
				settings, err := db.Instance.GetCleanSettings(event.chatID)
				if err != nil {
					c.Logger.Warn("auto-delete settings lookup failed", "chat_id", event.chatID)
					continue
				}

				if !settings.AutoDelete {
					continue
				}

				key := strconv.FormatInt(event.chatID, 10)
				group, ok := groupTypes.Get(key)
				if !ok {
					chat, err := c.GetChat(event.chatID)
					if err != nil {
						c.Logger.Warn("auto-delete chat lookup failed", "chat_id", event.chatID)
						continue
					}

					group = messaging.IsGroup(chat)
					groupTypes.Set(key, group)
				}
				if !group {
					continue
				}
				s.Track(event.chatID, event.oldID, event.id, event.at.Add(time.Duration(settings.Minutes)*time.Minute))
			}
		}
	}()
	return s, func() { cancel(); wg.Wait(); groupTypes.Close() }
}

// TelegramDeleted Cache eviction isn't a Telegram deletion and must not cancel cleanup.
func (s *Scheduler) TelegramDeleted(update *td.UpdateDeleteMessages) {
	if update != nil && !update.FromCache {
		s.Deleted(update.ChatId, update.MessageIds)
	}
}

func (s *Scheduler) TelegramMember(botID int64, u *td.UpdateChatMember) {
	if u == nil || u.NewChatMember == nil || u.ChatId >= 0 {
		return
	}
	sender, ok := u.NewChatMember.MemberId.(*td.MessageSenderUser)
	if !ok || sender.UserId != botID {
		return
	}

	switch status := u.NewChatMember.Status.(type) {
	case *td.ChatMemberStatusLeft, *td.ChatMemberStatusBanned:
		s.BlockChat(u.ChatId)
	case *td.ChatMemberStatusRestricted:
		if !status.IsMember {
			s.BlockChat(u.ChatId)
		} else {
			s.AllowChat(u.ChatId)
		}
	default:
		s.AllowChat(u.ChatId)
	}
}

func sentEvent(m *td.Message, oldID int64) (sent, bool) {
	if m == nil || !m.IsOutgoing || m.ChatId >= 0 || m.Id == 0 || m.SendingState != nil {
		return sent{}, false
	}
	return sent{chatID: m.ChatId, oldID: oldID, id: m.Id, at: time.Now()}, true
}
