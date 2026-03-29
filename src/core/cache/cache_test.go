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

func TestEvictExpiredRemovesExpiredEntries(t *testing.T) {
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

	if !getJanitor().has(m) {
		t.Error("mockCleaner not found in janitor")
	}
}

func TestJanitorUnregistration(t *testing.T) {
	m := &mockCleaner{}
	registerCache(m)

	if !getJanitor().has(m) {
		t.Fatal("mockCleaner not found in janitor before unregistration")
	}

	unregisterCache(m)

	if getJanitor().has(m) {
		t.Error("mockCleaner still found in janitor after unregistration")
	}
}

func TestJanitorUnregistrationOnClose(t *testing.T) {
	c := NewCache[string](time.Minute)
	if !getJanitor().has(c) {
		t.Fatal("cache not found in janitor after creation")
	}

	c.Close()

	if getJanitor().has(c) {
		t.Error("cache still found in janitor after Close()")
	}
}

func TestJanitorLifecycle(t *testing.T) {
	m := &mockCleaner{}
	registerCache(m)

	j := getJanitor()
	j.mu.Lock()
	running := j.running
	j.mu.Unlock()

	if !running {
		t.Error("expected janitor to be running after registration")
	}

	unregisterCache(m)

	j.mu.Lock()
	running = j.running
	count := len(j.caches)
	j.mu.Unlock()

	if count == 0 && running {
		t.Error("expected janitor to be stopped after all caches unregistered")
	}
}

func TestJanitorBackgroundCleanup(t *testing.T) {
	oldInterval := janitorInterval
	janitorInterval = 10 * time.Millisecond
	defer func() { janitorInterval = oldInterval }()

	m := &mockCleaner{}
	j := getJanitor()
	j.mu.Lock()
	for len(j.caches) > 0 {
		j.mu.Unlock()
		unregisterCache(j.caches[0])
		j.mu.Lock()
	}
	j.mu.Unlock()

	registerCache(m)
	defer unregisterCache(m)

	time.Sleep(100 * time.Millisecond)

	m.mu.Lock()
	evicted := m.evicted
	m.mu.Unlock()

	if !evicted {
		t.Error("expected janitor to have called evictExpired")
	}
}
