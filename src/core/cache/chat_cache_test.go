package cache

import (
	"ashokshau/tgmusic/src/utils"
	"sync"
	"testing"
)

// helpers

func makeTrack(id, name string) *utils.CachedTrack {
	return &utils.CachedTrack{
		TrackID:   id,
		Name:      name,
		URL:       "https://example.com/" + id,
		User:      "testuser",
		FilePath:  "/tmp/" + id + ".mp3",
		Thumbnail: "https://example.com/thumb/" + id,
		Duration:  180,
		Channel:   "TestChannel",
		Views:     "1000",
		IsVideo:   false,
		Platform:  "youtube",
	}
}

func newCache() *ChatCacher {
	return newChatCacher()
}

// AddSong

func TestAddSong_Single(t *testing.T) {
	c := newCache()
	n := c.AddSong(1, makeTrack("t1", "Track 1"))
	if n != 1 {
		t.Fatalf("expected queue length 1, got %d", n)
	}
}

func TestAddSong_Multiple(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	n := c.AddSong(1, makeTrack("t2", "Track 2"))
	if n != 2 {
		t.Fatalf("expected queue length 2, got %d", n)
	}
}

func TestAddSong_DifferentChats(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	n := c.AddSong(2, makeTrack("t2", "Track 2"))
	if n != 1 {
		t.Fatalf("expected queue length 1 for chat 2, got %d", n)
	}
}

// AddSongs

func TestAddSongs(t *testing.T) {
	c := newCache()
	tracks := []*utils.CachedTrack{
		makeTrack("t1", "Track 1"),
		makeTrack("t2", "Track 2"),
		makeTrack("t3", "Track 3"),
	}
	n := c.AddSongs(1, tracks)
	if n != 3 {
		t.Fatalf("expected 3, got %d", n)
	}
}

func TestAddSongs_AppendToExisting(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t0", "Track 0"))
	n := c.AddSongs(1, []*utils.CachedTrack{
		makeTrack("t1", "Track 1"),
		makeTrack("t2", "Track 2"),
	})
	if n != 3 {
		t.Fatalf("expected 3, got %d", n)
	}
}

// GetPlayingTrack

func TestGetPlayingTrack_Empty(t *testing.T) {
	c := newCache()
	if c.GetPlayingTrack(1) != nil {
		t.Fatal("expected nil for empty queue")
	}
}

func TestGetPlayingTrack_ReturnsFirst(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	c.AddSong(1, makeTrack("t2", "Track 2"))
	track := c.GetPlayingTrack(1)
	if track == nil {
		t.Fatal("expected track, got nil")
	}
	if track.TrackID != "t1" {
		t.Fatalf("expected t1, got %s", track.TrackID)
	}
}

// GetUpcomingTrack

func TestGetUpcomingTrack_Empty(t *testing.T) {
	c := newCache()
	if c.GetUpcomingTrack(1) != nil {
		t.Fatal("expected nil for empty queue")
	}
}

func TestGetUpcomingTrack_SingleTrack(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	if c.GetUpcomingTrack(1) != nil {
		t.Fatal("expected nil for single-track queue")
	}
}

func TestGetUpcomingTrack_ReturnsSecond(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	c.AddSong(1, makeTrack("t2", "Track 2"))
	track := c.GetUpcomingTrack(1)
	if track == nil {
		t.Fatal("expected track, got nil")
	}
	if track.TrackID != "t2" {
		t.Fatalf("expected t2, got %s", track.TrackID)
	}
}

// RemoveCurrentSong

func TestRemoveCurrentSong_Empty(t *testing.T) {
	c := newCache()
	if c.RemoveCurrentSong(1) != nil {
		t.Fatal("expected nil for empty queue")
	}
}

func TestRemoveCurrentSong_RemovesFirst(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	c.AddSong(1, makeTrack("t2", "Track 2"))

	removed := c.RemoveCurrentSong(1)
	if removed == nil || removed.TrackID != "t1" {
		t.Fatalf("expected t1 removed, got %v", removed)
	}

	playing := c.GetPlayingTrack(1)
	if playing == nil || playing.TrackID != "t2" {
		t.Fatalf("expected t2 now playing, got %v", playing)
	}
}

func TestRemoveCurrentSong_UntilEmpty(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	c.RemoveCurrentSong(1)

	if c.GetPlayingTrack(1) != nil {
		t.Fatal("expected nil after removing last track")
	}
	if c.IsActive(1) {
		t.Fatal("expected chat to be inactive after removing last track")
	}
}

// RemoveTrack

func TestRemoveTrack_ValidIndex(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	c.AddSong(1, makeTrack("t2", "Track 2"))
	c.AddSong(1, makeTrack("t3", "Track 3"))

	ok := c.RemoveTrack(1, 1) // remove t2
	if !ok {
		t.Fatal("expected RemoveTrack to return true")
	}
	if c.GetQueueLength(1) != 2 {
		t.Fatalf("expected queue length 2, got %d", c.GetQueueLength(1))
	}
	q := c.GetQueue(1)
	if q[0].TrackID != "t1" || q[1].TrackID != "t3" {
		t.Fatalf("unexpected queue order: %v", q)
	}
}

func TestRemoveTrack_InvalidIndex(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))

	if c.RemoveTrack(1, -1) {
		t.Fatal("expected false for negative index")
	}
	if c.RemoveTrack(1, 5) {
		t.Fatal("expected false for out-of-bounds index")
	}
}

func TestRemoveTrack_EmptyQueue(t *testing.T) {
	c := newCache()
	if c.RemoveTrack(1, 0) {
		t.Fatal("expected false for empty queue")
	}
}

// IsActive

func TestIsActive_False(t *testing.T) {
	c := newCache()
	if c.IsActive(1) {
		t.Fatal("expected inactive for unknown chat")
	}
}

func TestIsActive_True(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	if !c.IsActive(1) {
		t.Fatal("expected active after adding track")
	}
}

func TestIsActive_AfterClear(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	c.ClearChat(1)
	if c.IsActive(1) {
		t.Fatal("expected inactive after ClearChat")
	}
}

// ClearChat

func TestClearChat(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	c.AddSong(1, makeTrack("t2", "Track 2"))
	c.ClearChat(1)

	if c.GetQueueLength(1) != 0 {
		t.Fatal("expected queue length 0 after ClearChat")
	}
	if c.GetPlayingTrack(1) != nil {
		t.Fatal("expected nil playing track after ClearChat")
	}
}

func TestClearChat_NonExistent(t *testing.T) {
	c := newCache()
	c.ClearChat(999) // should not panic
}

// GetQueueLength

func TestGetQueueLength_Empty(t *testing.T) {
	c := newCache()
	if c.GetQueueLength(1) != 0 {
		t.Fatal("expected 0 for unknown chat")
	}
}

func TestGetQueueLength_AfterAdds(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	c.AddSong(1, makeTrack("t2", "Track 2"))
	if c.GetQueueLength(1) != 2 {
		t.Fatalf("expected 2, got %d", c.GetQueueLength(1))
	}
}

// Loop

func TestGetLoopCount_Empty(t *testing.T) {
	c := newCache()
	if c.GetLoopCount(1) != 0 {
		t.Fatal("expected 0 for empty queue")
	}
}

func TestSetAndGetLoopCount(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))

	ok := c.SetLoopCount(1, 5)
	if !ok {
		t.Fatal("expected SetLoopCount to return true")
	}
	if c.GetLoopCount(1) != 5 {
		t.Fatalf("expected loop count 5, got %d", c.GetLoopCount(1))
	}
}

func TestSetLoopCount_EmptyQueue(t *testing.T) {
	c := newCache()
	if c.SetLoopCount(1, 3) {
		t.Fatal("expected false for empty queue")
	}
}

// GetQueue

func TestGetQueue_Empty(t *testing.T) {
	c := newCache()
	if c.GetQueue(1) != nil {
		t.Fatal("expected nil for empty queue")
	}
}

func TestGetQueue_ReturnsCopy(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	c.AddSong(1, makeTrack("t2", "Track 2"))

	q := c.GetQueue(1)
	if len(q) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(q))
	}

	// mutating the returned slice should not affect the cache
	q[0] = makeTrack("hacked", "Hacked")
	if c.GetPlayingTrack(1).TrackID != "t1" {
		t.Fatal("GetQueue should return a copy, not the internal slice")
	}
}

// GetActiveChats

func TestGetActiveChats_Empty(t *testing.T) {
	c := newCache()
	if len(c.GetActiveChats()) != 0 {
		t.Fatal("expected no active chats")
	}
}

func TestGetActiveChats_Multiple(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	c.AddSong(2, makeTrack("t2", "Track 2"))
	c.AddSong(3, makeTrack("t3", "Track 3"))
	c.ClearChat(2)

	active := c.GetActiveChats()
	if len(active) != 2 {
		t.Fatalf("expected 2 active chats, got %d", len(active))
	}
	activeMap := make(map[int64]bool)
	for _, id := range active {
		activeMap[id] = true
	}
	if !activeMap[1] || !activeMap[3] {
		t.Fatalf("expected chats 1 and 3 to be active, got %v", active)
	}
}

// GetTrackIfExists

func TestGetTrackIfExists_Found(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	c.AddSong(1, makeTrack("t2", "Track 2"))

	track := c.GetTrackIfExists(1, "t2")
	if track == nil || track.TrackID != "t2" {
		t.Fatalf("expected t2, got %v", track)
	}
}

func TestGetTrackIfExists_NotFound(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))

	if c.GetTrackIfExists(1, "nope") != nil {
		t.Fatal("expected nil for missing track")
	}
}

func TestGetTrackIfExists_UnknownChat(t *testing.T) {
	c := newCache()
	if c.GetTrackIfExists(999, "t1") != nil {
		t.Fatal("expected nil for unknown chat")
	}
}

// Concurrency

func TestConcurrentAddAndRead(t *testing.T) {
	c := newCache()
	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := int64(i % 5)
			c.AddSong(id, makeTrack("t", "Track"))
			c.GetPlayingTrack(id)
			c.IsActive(id)
			c.GetQueueLength(id)
		}(i)
	}
	wg.Wait()
}

func TestConcurrentAddAndRemove(t *testing.T) {
	c := newCache()
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.AddSong(1, makeTrack("t", "Track"))
		}()
	}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.RemoveCurrentSong(1)
		}()
	}
	wg.Wait()
	_ = c.GetQueueLength(1)
}

func TestConcurrentIsActiveAndClear(t *testing.T) {
	c := newCache()
	c.AddSong(1, makeTrack("t1", "Track 1"))
	var wg sync.WaitGroup

	for i := 0; i < 30; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.IsActive(1)
		}()
		go func() {
			defer wg.Done()
			c.ClearChat(1)
			c.AddSong(1, makeTrack("t1", "Track 1"))
		}()
	}
	wg.Wait()
}
