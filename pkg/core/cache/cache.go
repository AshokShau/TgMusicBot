package cache

import (
	"sync"
	"time"
)

// CacheItem represents an item stored in the cache, containing a value and its expiration time.
type CacheItem[T any] struct {
	Value      T
	Expiration time.Time
}

// Cache is a generic, thread-safe TTL cache that stores values with string keys.
type Cache[T any] struct {
	data map[string]CacheItem[T]
	mu   sync.RWMutex
	ttl  time.Duration
}

// NewCache initializes and returns a new Cache with a specified default TTL.
// The ttl parameter sets the default time-to-live duration for cache items.
func NewCache[T any](ttl time.Duration) *Cache[T] {
	return &Cache[T]{
		data: make(map[string]CacheItem[T]),
		ttl:  ttl,
	}
}

// Get retrieves a value from the cache by its key.
// It returns the cached value and true if the key exists and has not expired; otherwise, it returns the zero value and false.
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	item, ok := c.data[key]
	c.mu.RUnlock()

	if !ok || time.Now().After(item.Expiration) {
		var zero T
		return zero, false
	}
	return item.Value, true
}

// Set adds or updates a value in the cache with the default TTL.
// It takes a key and a value to store.
func (c *Cache[T]) Set(key string, value T) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL adds or updates a value in the cache with a custom TTL, overriding the default.
// It takes a key, a value, and a custom TTL duration.
func (c *Cache[T]) SetWithTTL(key string, value T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = CacheItem[T]{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
}

// Delete removes an item from the cache by its key.
func (c *Cache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// Clear purges all items from the cache, making it empty.
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]CacheItem[T])
}
