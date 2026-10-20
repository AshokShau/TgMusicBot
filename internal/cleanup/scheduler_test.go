package cleanup

import (
	"context"
	"errors"
	td "github.com/AshokShau/gotdbot"
	"sync"
	"testing"
	"time"
)

func scheduler() *Scheduler {
	return New(func(int64, []int64) error { return nil }, func(int64) bool { return true })
}
func TestReplacementAndDuplicate(t *testing.T) {
	s := scheduler()
	due := time.Now().Add(2 * time.Minute)
	s.Track(-1, 0, 11, due)
	s.Track(-1, 11, 22, due.Add(time.Minute))
	s.Track(-1, 11, 22, due.Add(5*time.Minute))
	if len(s.pending) != 1 || len(s.queue) != 1 || !s.pending[Key{-1, 22}].due.Equal(due) {
		t.Fatal("replacement or duplicate changed expiration")
	}
	if len(s.take(due.Add(-time.Nanosecond))) != 0 {
		t.Fatal("deleted before expiration")
	}
	batch := s.take(due)
	if len(batch) != 1 || batch[0].MessageID != 22 {
		t.Fatal("temporary ID reached worker")
	}
	if len(s.take(due)) != 0 {
		t.Fatal("in-flight message claimed twice")
	}
	s.finish(batch, nil, due)
	if s.Track(-1, 0, 22, due) || len(s.pending) != 0 {
		t.Fatal("late duplicate resurrected job")
	}
}
func TestManualDeleteAndChatIsolation(t *testing.T) {
	s := scheduler()
	now := time.Now()
	s.Track(-1, 0, 1, now)
	s.Track(-2, 0, 1, now)
	batch := s.take(now)
	s.Deleted(-1, []int64{1, 1})
	active := s.active(batch)
	if len(active) != 1 || active[0].ChatID != -2 {
		t.Fatal("manual deletion crossed chat boundary")
	}
	s.finish(batch, errors.New("network"), now)
	if len(s.pending) != 0 {
		t.Fatal("failed or manually deleted message was retained")
	}
	s.Deleted(-3, []int64{9})
	if s.Track(-3, 0, 9, now) {
		t.Fatal("out-of-order update resurrected deletion")
	}
	if s.Track(1, 0, 1, now) {
		t.Fatal("private chat tracked")
	}
}
func TestBulkDeletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	seen := map[Key]bool{}
	calls := 0
	s := New(func(chatID int64, ids []int64) error {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if len(ids) > 100 || len(ids) == 0 {
			t.Errorf("invalid batch size %d", len(ids))
		}
		for _, id := range ids {
			if (chatID == -1 && id > 205) || (chatID == -2 && id <= 205) {
				t.Error("mixed chats")
			}
			k := Key{chatID, id}
			if seen[k] {
				t.Error("duplicate deletion")
			}
			seen[k] = true
		}
		if len(seen) == 208 {
			cancel()
		}
		return nil
	}, func(int64) bool { return true })
	now := time.Now()
	for id := int64(1); id <= 208; id++ {
		chatID := int64(-1)
		if id > 205 {
			chatID = -2
		}
		s.Track(chatID, 0, id, now)
	}
	s.Track(-1, 0, 999, now.Add(time.Hour))
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("scheduler stalled")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 208 || calls < 4 || calls > 5 {
		t.Fatalf("count=%d calls=%d", len(seen), calls)
	}
	if seen[Key{-1, 999}] {
		t.Fatal("deleted future job")
	}
}
func TestDeletionErrorsClearJobsAndRespectFloodWait(t *testing.T) {
	for _, err := range []error{errors.New("network"), &td.Error{Code: 400, Message: "MESSAGE_ID_INVALID"}, &td.Error{Code: 403, Message: "Forbidden"}, &td.Error{Code: 429, Message: "Too Many Requests: retry after 90"}} {
		s := scheduler()
		now := time.Now()
		s.Track(-1, 0, 1, now)
		s.finish(s.take(now), err, now)
		if len(s.pending) != 0 || len(s.queue) != 0 {
			t.Fatal("failed job was retried or retained")
		}
		if te, ok := err.(*td.Error); ok && te.Code == 429 {
			s.Track(-2, 0, 2, now)
			if len(s.take(now.Add(89*time.Second))) != 0 {
				t.Fatal("later batch ignored flood wait")
			}
			if len(s.take(now.Add(90*time.Second))) != 1 {
				t.Fatal("flood wait did not expire")
			}
		}
	}
}
func TestConcurrentScheduling(t *testing.T) {
	s := scheduler()
	now := time.Now()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for id := int64(1); id <= 500; id++ {
				s.Track(-1, 0, id, now)
				if id%3 == 0 {
					s.Deleted(-1, []int64{id})
				}
				batch := s.take(now)
				s.finish(batch, nil, now)
			}
		})
	}
	wg.Wait()
	for _, j := range s.take(now) {
		s.Deleted(j.ChatID, []int64{j.MessageID})
	}
	if len(s.pending) != 0 {
		t.Fatal("stale concurrent entries")
	}
}
func TestCancelAndBoundedQueue(t *testing.T) {
	s := scheduler()
	now := time.Now().Add(time.Hour)
	for id := int64(1); id <= MaxPending; id++ {
		if !s.Track(-1, 0, id, now) {
			t.Fatal("premature capacity rejection")
		}
	}
	if s.Track(-2, 0, 1, now) {
		t.Fatal("queue grew beyond capacity")
	}
	s.CancelChat(-1)
	if len(s.pending) != 0 || len(s.queue) != 0 || len(s.removed) > MaxPending {
		t.Fatal("queue or tombstones leaked")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.Run(ctx)
}

func TestTelegramDeletionUpdates(t *testing.T) {
	s := scheduler()
	now := time.Now()
	s.Track(-1, 0, 1, now)
	s.TelegramDeleted(&td.UpdateDeleteMessages{ChatId: -1, MessageIds: []int64{1}, FromCache: true})
	if len(s.pending) != 1 {
		t.Fatal("cache eviction canceled deletion")
	}
	s.TelegramDeleted(&td.UpdateDeleteMessages{ChatId: -1, MessageIds: []int64{1}, IsPermanent: true})
	s.TelegramDeleted(&td.UpdateDeleteMessages{ChatId: -1, MessageIds: []int64{1}, IsPermanent: true})
	if len(s.pending) != 0 || len(s.queue) != 0 {
		t.Fatal("deleted message retained")
	}
}

func TestBotKickedClearsOnlyItsChat(t *testing.T) {
	s := scheduler()
	now := time.Now()
	s.Track(-1, 0, 1, now)
	s.Track(-2, 0, 1, now)
	update := func(userID int64, status td.ChatMemberStatus) *td.UpdateChatMember {
		return &td.UpdateChatMember{ChatId: -1, NewChatMember: &td.ChatMember{MemberId: &td.MessageSenderUser{UserId: userID}, Status: status}}
	}
	s.TelegramMember(99, update(1, &td.ChatMemberStatusLeft{}))
	if len(s.pending) != 2 {
		t.Fatal("another user's departure cleared bot jobs")
	}
	s.TelegramMember(99, update(99, &td.ChatMemberStatusBanned{}))
	if len(s.pending) != 1 || s.pending[Key{-2, 1}] == nil {
		t.Fatal("kick failed to isolate cleanup")
	}
	if s.Track(-1, 0, 2, now) {
		t.Fatal("delayed send update resurrected kicked chat")
	}
	s.TelegramMember(99, update(99, &td.ChatMemberStatusMember{}))
	if !s.Track(-1, 0, 3, now) {
		t.Fatal("rejoined chat remained blocked")
	}
	s.TelegramMember(99, update(99, &td.ChatMemberStatusLeft{}))
	if s.pending[Key{-1, 3}] != nil {
		t.Fatal("bot departure retained jobs")
	}
}

func TestOnlyFinalOutgoingGroupMessagesAreTracked(t *testing.T) {
	for _, m := range []*td.Message{nil, {ChatId: 1, Id: 1, IsOutgoing: true}, {ChatId: -1, Id: 1}, {ChatId: -1, Id: 1, IsOutgoing: true, SendingState: &td.MessageSendingStatePending{}}} {
		if _, ok := sentEvent(m, 0); ok {
			t.Fatal("private, incoming or provisional message accepted")
		}
	}
	event, ok := sentEvent(&td.Message{ChatId: -1, Id: 22, IsOutgoing: true}, 11)
	if !ok || event.chatID != -1 || event.oldID != 11 || event.id != 22 || event.at.IsZero() {
		t.Fatal("final message metadata lost")
	}
}
func TestSchedulerWakesForEarlierJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := make(chan int64, 1)
	s := New(func(chatID int64, ids []int64) error { calls <- ids[0]; cancel(); return nil }, func(int64) bool { return true })
	s.Track(-1, 0, 1, time.Now().Add(time.Hour))
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	s.Track(-1, 0, 2, time.Now())
	select {
	case id := <-calls:
		if id != 2 {
			t.Fatal("future job deleted")
		}
	case <-time.After(time.Second):
		t.Fatal("earlier deadline didn't wake scheduler")
	}
	<-done
}
func TestFloodWaitDefersOtherChatsWithoutRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	calls := 0
	s := New(func(int64, []int64) error {
		calls++
		return &td.Error{Code: 429, Message: "Too Many Requests: retry after 30"}
	}, func(int64) bool { return true })
	now := time.Now()
	s.Track(-1, 0, 1, now)
	s.Track(-2, 0, 1, now)
	s.Run(ctx)
	if calls != 1 || len(s.pending) != 1 || len(s.queue) != 1 {
		t.Fatalf("calls=%d pending=%d queue=%d", calls, len(s.pending), len(s.queue))
	}
}
