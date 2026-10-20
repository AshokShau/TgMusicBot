package cleanup

import (
	"container/heap"
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	td "github.com/AshokShau/gotdbot"
)

const MaxPending = 100000
const BatchSize = 100

type Key struct{ ChatID, MessageID int64 }
type job struct {
	Key
	due   time.Time
	index int
}
type jobs []*job

func (h jobs) Len() int           { return len(h) }
func (h jobs) Less(i, j int) bool { return h[i].due.Before(h[j].due) }
func (h jobs) Swap(i, j int)      { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *jobs) Push(v any)        { j := v.(*job); j.index = len(*h); *h = append(*h, j) }
func (h *jobs) Pop() any {
	old := *h
	j := old[len(old)-1]
	old[len(old)-1] = nil
	*h = old[:len(old)-1]
	j.index = -1
	return j
}

type Scheduler struct {
	mu      sync.Mutex
	pending map[Key]*job
	queue   jobs
	// Recent removals prevent delayed/duplicate send updates resurrecting jobs.
	removed     map[Key]struct{}
	removals    []Key
	removalHead int
	running     atomic.Bool
	wake        chan struct{}
	delete      func(int64, []int64) error
	enabled     func(int64) bool
	floodUntil  time.Time
}

func New(deleteFn func(int64, []int64) error, enabled func(int64) bool) *Scheduler {
	return &Scheduler{pending: make(map[Key]*job), removed: make(map[Key]struct{}), wake: make(chan struct{}, 1), delete: deleteFn, enabled: enabled}
}
func (s *Scheduler) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
func (s *Scheduler) remember(k Key) {
	if _, ok := s.removed[k]; ok {
		return
	}
	if len(s.removals) < MaxPending {
		s.removals = append(s.removals, k)
	} else {
		delete(s.removed, s.removals[s.removalHead])
		s.removals[s.removalHead] = k
		s.removalHead = (s.removalHead + 1) % MaxPending
	}
	s.removed[k] = struct{}{}

}
func (s *Scheduler) forget(k Key) {
	if j := s.pending[k]; j != nil {
		if j.index >= 0 {
			heap.Remove(&s.queue, j.index)
		}
		delete(s.pending, k)
	}
	s.remember(k)
}

// Track is idempotent: duplicate updates cannot reset a timer. Temporary IDs
// are replaced atomically, preserving the original expiration when present.
func (s *Scheduler) Track(chatID, oldID, finalID int64, due time.Time) bool {
	if chatID >= 0 || finalID == 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := Key{chatID, finalID}
	replaced := false
	if _, blocked := s.removed[Key{chatID, 0}]; blocked {
		return false
	}
	if oldID != 0 && oldID != finalID {
		old := Key{chatID, oldID}
		if j := s.pending[old]; j != nil {
			due = j.due
			replaced = true
			s.forget(old)
		}
	}
	if _, ok := s.removed[k]; ok {
		return false
	}
	if existing, ok := s.pending[k]; ok {
		if replaced && due.Before(existing.due) {
			existing.due = due
			if existing.index >= 0 {
				heap.Fix(&s.queue, existing.index)
			}
		}
		return true
	}

	if len(s.pending) >= MaxPending {
		slog.Warn("auto-delete queue at capacity", "capacity", MaxPending)
		return false
	}

	j := &job{Key: k, due: due}
	s.pending[k] = j
	heap.Push(&s.queue, j)
	s.signal()
	return true
}
func (s *Scheduler) Deleted(chatID int64, ids []int64) {
	if chatID >= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		s.forget(Key{chatID, id})
	}
	s.signal()
}
func (s *Scheduler) cancelChatLocked(chatID int64) {
	for k := range s.pending {
		if k.ChatID == chatID {
			s.forget(k)
		}
	}
}
func (s *Scheduler) CancelChat(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelChatLocked(chatID)
	s.signal()
}

// BlockChat also rejects send updates already queued when the bot was kicked.
func (s *Scheduler) BlockChat(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelChatLocked(chatID)
	s.remember(Key{chatID, 0})
	s.signal()
}
func (s *Scheduler) AllowChat(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.removed, Key{chatID, 0})
}

// take claims a bounded number of due jobs under the lock. Jobs stay in the
// map while in flight, allowing a manual delete to cancel them before the call.
func (s *Scheduler) take(now time.Time) []*job {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.Before(s.floodUntil) {
		return nil
	}
	batch := make([]*job, 0, BatchSize*10)
	for len(s.queue) > 0 && len(batch) < BatchSize*10 && !s.queue[0].due.After(now) {
		j := heap.Pop(&s.queue).(*job)
		batch = append(batch, j)
	}
	return batch
}

func (s *Scheduler) active(batch []*job) []*job {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*job, 0, len(batch))
	for _, j := range batch {
		if s.pending[j.Key] == j {
			result = append(result, j)
		}
	}
	return result
}

// A failed deletion is terminal for this batch. Flood waits pause later
// batches without retrying failed jobs.
func (s *Scheduler) finish(batch []*job, err error, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range batch {
		if s.pending[j.Key] == j {
			s.forget(j.Key)
		}
	}
	var te *td.Error
	if errors.As(err, &te) && te.Code == 429 {
		seconds := te.GetRetryAfter()
		if seconds < 1 {
			seconds = 60
		}
		s.floodUntil = now.Add(time.Duration(seconds) * time.Second)
	}
}

func (s *Scheduler) nextDelay(now time.Time) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.Before(s.floodUntil) {
		return s.floodUntil.Sub(now)
	}
	if len(s.queue) == 0 {
		return time.Hour
	}
	d := s.queue[0].due.Sub(now)
	if d < 0 {
		return 0
	}
	return d
}

func (s *Scheduler) Run(ctx context.Context) {
	if !s.running.CompareAndSwap(false, true) {
		return
	}
	defer s.running.Store(false)
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
		case <-timer.C:
		}
		if ctx.Err() != nil {
			return
		}

		// Collect due jobs in a bounded turn, then batch by chat.
		groups := make(map[int64][]*job)
		for _, j := range s.take(time.Now()) {
			groups[j.ChatID] = append(groups[j.ChatID], j)
		}
		batches := make([][]*job, 0, len(groups))
		for _, group := range groups {
			for len(group) > 0 {
				n := min(BatchSize, len(group))
				batches = append(batches, group[:n])
				group = group[n:]
			}
		}
		for _, batch := range batches {
			chatID := batch[0].ChatID
			if ctx.Err() != nil {
				return
			}
			if !s.enabled(chatID) {
				s.CancelChat(chatID)
				continue
			}
			batch = s.active(batch)
			if len(batch) == 0 {
				continue
			}
			// A flood wait from a previous chat defers all remaining chats.
			s.mu.Lock()
			blocked := time.Now().Before(s.floodUntil)
			until := s.floodUntil
			if blocked {
				for _, j := range batch {
					if s.pending[j.Key] == j {
						j.due = until
						heap.Push(&s.queue, j)
					}
				}
			}
			s.mu.Unlock()
			if blocked {
				continue
			}
			ids := make([]int64, len(batch))
			for i, j := range batch {
				ids[i] = j.MessageID
			}
			err := s.delete(chatID, ids)
			if err != nil {
				slog.Warn("auto-delete request failed", "chat_id", chatID, "count", len(ids))
			}
			s.finish(batch, err, time.Now())
			// Pace successful calls as well as retries; no burst loop across groups.
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(s.nextDelay(time.Now()))
	}
}
