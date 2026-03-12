/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package cache

import (
	"sync"
	"time"
)

// Item holds a cached value and its expiration time.
type Item[T any] struct {
	Value      T
	Expiration time.Time
}

// Cache is a generic, thread-safe TTL cache with automatic background eviction.
type Cache[T any] struct {
	mu   sync.RWMutex
	data map[string]Item[T]
	ttl  time.Duration
	stop chan struct{}
}

// NewCache creates a Cache with the given default TTL and starts a background
// goroutine that evicts expired entries every cleanupInterval.
// Call Close() when the cache is no longer needed to stop the goroutine.
func NewCache[T any](ttl time.Duration) *Cache[T] {
	// Cleanup runs at half the TTL (at least every 30 s) so entries are
	// evicted reasonably promptly without hammering the lock.
	cleanupInterval := ttl / 2
	if cleanupInterval < 30*time.Second {
		cleanupInterval = 30 * time.Second
	}

	c := &Cache[T]{
		data: make(map[string]Item[T]),
		ttl:  ttl,
		stop: make(chan struct{}),
	}
	go c.runCleanup(cleanupInterval)
	return c
}

// Get returns the value for key and true if it exists and has not expired.
// Expired entries are deleted on access (lazy eviction in addition to background sweep).
func (c *Cache[T]) Get(key string) (T, bool) {
	// Fast path: read lock.
	c.mu.RLock()
	item, ok := c.data[key]
	c.mu.RUnlock()

	if !ok {
		var zero T
		return zero, false
	}

	if time.Now().After(item.Expiration) {
		// Lazy delete: upgrade to write lock and evict.
		c.mu.Lock()
		// Re-check under write lock; another goroutine may have refreshed it.
		if it, still := c.data[key]; still && time.Now().After(it.Expiration) {
			delete(c.data, key)
		}
		c.mu.Unlock()

		var zero T
		return zero, false
	}

	return item.Value, true
}

// Set stores value under key using the default TTL.
func (c *Cache[T]) Set(key string, value T) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores value under key with a custom TTL.
func (c *Cache[T]) SetWithTTL(key string, value T, ttl time.Duration) {
	c.mu.Lock()
	c.data[key] = Item[T]{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}

// Delete removes key from the cache immediately.
func (c *Cache[T]) Delete(key string) {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
}

// Clear evicts all items at once.
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	c.data = make(map[string]Item[T])
	c.mu.Unlock()
}

// Size returns the number of entries currently in the cache (including not-yet-evicted expired ones).
func (c *Cache[T]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Close stops the background cleanup goroutine.
// The cache remains usable after Close; only background eviction stops.
func (c *Cache[T]) Close() {
	close(c.stop)
}

// runCleanup periodically scans and removes expired entries.
func (c *Cache[T]) runCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.evictExpired()
		case <-c.stop:
			return
		}
	}
}

// evictExpired removes all entries whose TTL has elapsed.
func (c *Cache[T]) evictExpired() {
	now := time.Now()
	c.mu.Lock()
	for key, item := range c.data {
		if now.After(item.Expiration) {
			delete(c.data, key)
		}
	}
	c.mu.Unlock()
}
