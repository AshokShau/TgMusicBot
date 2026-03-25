/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package cache

import (
	"ashokshau/tgmusic/src/utils"
	"sync"
)

// ChatData holds the state of a chat's music queue.
type ChatData struct {
	Queue []*utils.CachedTrack
}

// ChatCacher is a thread-safe cache that manages music queues for multiple chats.
type ChatCacher struct {
	mu        sync.RWMutex
	chatCache map[int64]*ChatData
}

// newChatCacher initializes and returns a new ChatCacher.
func newChatCacher() *ChatCacher {
	return &ChatCacher{
		chatCache: make(map[int64]*ChatData),
	}
}

// getOrCreate returns the ChatData for a chat, creating it if absent.
// Caller must hold the write lock.
func (c *ChatCacher) getOrCreate(chatID int64) *ChatData {
	data, ok := c.chatCache[chatID]
	if !ok {
		data = &ChatData{}
		c.chatCache[chatID] = data
	}
	return data
}

// AddSong adds a track to a chat's queue and returns the new queue length.
func (c *ChatCacher) AddSong(chatID int64, song *utils.CachedTrack) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	data := c.getOrCreate(chatID)
	data.Queue = append(data.Queue, song)
	return len(data.Queue)
}

// AddSongs appends multiple tracks to a chat's queue and returns the new queue length.
func (c *ChatCacher) AddSongs(chatID int64, songs []*utils.CachedTrack) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	data := c.getOrCreate(chatID)
	data.Queue = append(data.Queue, songs...)
	return len(data.Queue)
}

// GetPlayingTrack returns the first track in the queue, or nil if empty.
func (c *ChatCacher) GetPlayingTrack(chatID int64) *utils.CachedTrack {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	if !ok || len(data.Queue) == 0 {
		return nil
	}
	return data.Queue[0]
}

// GetUpcomingTrack returns the second track in the queue, or nil if fewer than two tracks exist.
func (c *ChatCacher) GetUpcomingTrack(chatID int64) *utils.CachedTrack {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	if !ok || len(data.Queue) < 2 {
		return nil
	}
	return data.Queue[1]
}

// RemoveCurrentSong removes and returns the currently playing track, or nil if the queue is empty.
func (c *ChatCacher) RemoveCurrentSong(chatID int64) *utils.CachedTrack {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, ok := c.chatCache[chatID]
	if !ok || len(data.Queue) == 0 {
		return nil
	}

	removed := data.Queue[0]
	data.Queue[0] = nil
	data.Queue = data.Queue[1:]
	return removed
}

// RemoveTrack removes the track at the given index and returns whether it succeeded.
func (c *ChatCacher) RemoveTrack(chatID int64, index int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, ok := c.chatCache[chatID]
	if !ok || index < 0 || index >= len(data.Queue) {
		return false
	}

	q := data.Queue
	copy(q[index:], q[index+1:])
	q[len(q)-1] = nil
	data.Queue = q[:len(q)-1]
	return true
}

// IsActive returns true if the chat has at least one queued track.
func (c *ChatCacher) IsActive(chatID int64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	return ok && len(data.Queue) > 0
}

// ClearChat deletes all queued tracks for a chat.
func (c *ChatCacher) ClearChat(chatID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if data, ok := c.chatCache[chatID]; ok {
		for i := range data.Queue {
			data.Queue[i] = nil
		}
		delete(c.chatCache, chatID)
	}
}

// GetQueueLength returns the number of tracks queued for a chat.
func (c *ChatCacher) GetQueueLength(chatID int64) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	if !ok {
		return 0
	}
	return len(data.Queue)
}

// GetLoopCount returns the loop count of the currently playing track, or 0 if none.
func (c *ChatCacher) GetLoopCount(chatID int64) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	if !ok || len(data.Queue) == 0 {
		return 0
	}
	return data.Queue[0].Loop
}

// SetLoopCount sets the loop count on the currently playing track.
// Returns false if there is no active track.
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

// GetQueue returns a shallow copy of the queue for a chat.
func (c *ChatCacher) GetQueue(chatID int64) []*utils.CachedTrack {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.chatCache[chatID]
	if !ok || len(data.Queue) == 0 {
		return nil
	}
	return append([]*utils.CachedTrack(nil), data.Queue...)
}

// GetActiveChats returns the IDs of all chats with at least one queued track.
func (c *ChatCacher) GetActiveChats() []int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	active := make([]int64, 0, len(c.chatCache))
	for chatID, data := range c.chatCache {
		if len(data.Queue) > 0 {
			active = append(active, chatID)
		}
	}
	return active
}

// GetTrackIfExists searches the queue for a track by ID and returns it, or nil if not found.
func (c *ChatCacher) GetTrackIfExists(chatID int64, trackID string) *utils.CachedTrack {
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

// ChatCache is the global instance.
var ChatCache = newChatCacher()
