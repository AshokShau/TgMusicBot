package cache

import (
	"os"
	"path/filepath"
	"sync"
)

// ChatData holds the state of a chat's music queue, including whether it is active and the list of tracks.
type ChatData struct {
	IsActive bool
	Queue    []*CachedTrack
}

// ChatCacher is a thread-safe cache that manages music queues for multiple chats.
type ChatCacher struct {
	mu        sync.RWMutex
	chatCache map[int64]*ChatData
}

// NewChatCacher initializes and returns a new ChatCacher.
func NewChatCacher() *ChatCacher {
	return &ChatCacher{
		chatCache: make(map[int64]*ChatData),
	}
}

// AddSong adds a new song to a chat's queue. If the chat does not exist, it creates a new one.
// It takes a chat ID and a CachedTrack to add, and returns the added track.
func (c *ChatCacher) AddSong(chatID int64, song *CachedTrack) *CachedTrack {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, ok := c.chatCache[chatID]
	if !ok {
		data = &ChatData{IsActive: true, Queue: []*CachedTrack{}}
		c.chatCache[chatID] = data
	}

	data.Queue = append(data.Queue, song)
	return song
}

// GetUpcomingTrack retrieves the next song in the queue for a given chat.
// It returns the upcoming track or nil if the queue is empty or has only one song.
func (c *ChatCacher) GetUpcomingTrack(chatID int64) *CachedTrack {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	if !ok || len(data.Queue) < 2 {
		return nil
	}
	return data.Queue[1]
}

// GetPlayingTrack retrieves the currently playing song for a given chat.
// It returns the current track or nil if the queue is empty.
func (c *ChatCacher) GetPlayingTrack(chatID int64) *CachedTrack {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	if !ok || len(data.Queue) == 0 {
		return nil
	}
	return data.Queue[0]
}

// RemoveCurrentSong removes the currently playing song from the queue.
// It can also optionally clear the associated file from the disk.
// It returns the removed track or nil if the queue was empty.
func (c *ChatCacher) RemoveCurrentSong(chatID int64, diskClear bool) *CachedTrack {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, ok := c.chatCache[chatID]
	if !ok || len(data.Queue) == 0 {
		return nil
	}

	removed := data.Queue[0]
	data.Queue = data.Queue[1:]

	if diskClear && removed.FilePath != "" {
		_ = os.Remove(removed.FilePath)
		_ = os.Remove(filepath.Join("database", "photos", removed.TrackID+".png"))
	}

	return removed
}

// IsActive checks if the music player is currently active in a specific chat.
// It returns true if active, otherwise false.
func (c *ChatCacher) IsActive(chatID int64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	return ok && data.IsActive
}

// SetActive updates the active state of the music player for a chat.
func (c *ChatCacher) SetActive(chatID int64, active bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, ok := c.chatCache[chatID]
	if !ok {
		data = &ChatData{Queue: []*CachedTrack{}}
		c.chatCache[chatID] = data
	}
	data.IsActive = active
}

// ClearChat removes all tracks from a chat's queue and optionally deletes the files from disk.
func (c *ChatCacher) ClearChat(chatID int64, diskClear bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, ok := c.chatCache[chatID]
	if !ok {
		return
	}

	if diskClear {
		for _, track := range data.Queue {
			if track.FilePath != "" {
				_ = os.Remove(track.FilePath)
			}
		}
	}
	delete(c.chatCache, chatID)
}

// GetQueueLength returns the total number of songs in a chat's queue.
func (c *ChatCacher) GetQueueLength(chatID int64) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	if !ok {
		return 0
	}
	return len(data.Queue)
}

// GetLoopCount retrieves the loop count for the currently playing song in a chat.
func (c *ChatCacher) GetLoopCount(chatID int64) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	if !ok || len(data.Queue) == 0 {
		return 0
	}
	return data.Queue[0].Loop
}

// SetLoopCount sets the loop count for the currently playing song.
// It returns true if the loop count was successfully set, otherwise false.
func (c *ChatCacher) SetLoopCount(chatID int64, loop int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, ok := c.chatCache[chatID]
	if !ok || len(data.Queue) == 0 {
		return false
	}
	data.Queue[0].Loop = loop
	return true
}

// RemoveTrack removes a specific song from the queue by its index.
// It returns true if the track was successfully removed, otherwise false.
func (c *ChatCacher) RemoveTrack(chatID int64, index int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, ok := c.chatCache[chatID]
	if !ok || index < 0 || index >= len(data.Queue) {
		return false
	}

	data.Queue = append(data.Queue[:index], data.Queue[index+1:]...)
	return true
}

// GetQueue returns a copy of the current song queue for a chat.
func (c *ChatCacher) GetQueue(chatID int64) []*CachedTrack {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	if !ok {
		return []*CachedTrack{}
	}
	return append([]*CachedTrack(nil), data.Queue...)
}

// GetActiveChats returns a list of all chat IDs where the music player is currently active.
func (c *ChatCacher) GetActiveChats() []int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var active []int64
	for chatID, data := range c.chatCache {
		if data.IsActive {
			active = append(active, chatID)
		}
	}
	return active
}

// GetTrackIfExists searches for a track in the queue by its ID and returns it if found.
// It returns the track or nil if it does not exist in the queue.
func (c *ChatCacher) GetTrackIfExists(chatID int64, trackID string) *CachedTrack {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	if !ok {
		return nil
	}

	for _, t := range data.Queue {
		if t.TrackID == trackID {
			return t
		}
	}
	return nil
}

// ChatCache is the global chat cacher.
var ChatCache = NewChatCacher()
