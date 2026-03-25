package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := NewCache[string](time.Second)
	defer c.Close()

	c.Set("key1", "value1")
	val, ok := c.Get("key1")
	if !ok || val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}
}

func TestCacheExpiration(t *testing.T) {
	c := NewCache[string](10 * time.Millisecond)
	defer c.Close()

	c.Set("key1", "value1")
	time.Sleep(20 * time.Millisecond)

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected key1 to be expired")
	}
}

func TestJanitorEviction(t *testing.T) {
	c := NewCache[string](10 * time.Millisecond)
	defer c.Close()

	c.Set("key1", "value1")
	time.Sleep(20 * time.Millisecond)

	c.evictExpired()

	if c.Size() != 0 {
		t.Errorf("expected cache size 0, got %d", c.Size())
	}
}

type mockCleaner struct {
	evicted bool
	mu      sync.Mutex
}

func (m *mockCleaner) evictExpired() {
	m.mu.Lock()
	m.evicted = true
	m.mu.Unlock()
}

func TestJanitorRegistration(t *testing.T) {
	m := &mockCleaner{}
	registerCache(m)
	defer unregisterCache(m)

	found := false
	sharedJanitor.mu.Lock()
	for _, c := range sharedJanitor.caches {
		if c == m {
			found = true
			break
		}
	}
	sharedJanitor.mu.Unlock()

	if !found {
		t.Error("mockCleaner not found in janitor")
	}
}
