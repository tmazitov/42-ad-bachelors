// Package cache holds a small in-memory cache with expiring entries, used to
// avoid repeating 42 API calls.
package cache

import (
	"sync"
	"time"
)

// TTL maps keys to values that expire after a fixed time. It holds at most
// maxEntries; when full, expired entries are dropped first, then the ones
// closest to expiring. It is safe for concurrent use.
type TTL[K comparable, V any] struct {
	ttl        time.Duration
	maxEntries int
	now        func() time.Time

	mu      sync.Mutex
	entries map[K]ttlEntry[V]
}

type ttlEntry[V any] struct {
	value   V
	expires time.Time
}

// NewTTL returns an empty cache whose entries live for ttl.
func NewTTL[K comparable, V any](ttl time.Duration, maxEntries int) *TTL[K, V] {
	return &TTL[K, V]{ttl: ttl, maxEntries: maxEntries, now: time.Now, entries: map[K]ttlEntry[V]{}}
}

// Get returns the value stored for key if it has not expired.
func (c *TTL[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || !c.now().Before(e.expires) {
		var zero V
		return zero, false
	}
	return e.value, true
}

// Set stores value for key for the cache's TTL.
func (c *TTL[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if _, exists := c.entries[key]; !exists && len(c.entries) >= c.maxEntries {
		c.evict(now)
	}
	c.entries[key] = ttlEntry[V]{value: value, expires: now.Add(c.ttl)}
}

// Delete removes key.
func (c *TTL[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Len returns the number of stored entries, expired or not.
func (c *TTL[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// evict makes room for one entry: it drops expired entries, or else the one
// expiring first (the oldest, since all share one TTL).
func (c *TTL[K, V]) evict(now time.Time) {
	var oldest K
	var oldestAt time.Time
	first := true
	for k, e := range c.entries {
		if !now.Before(e.expires) {
			delete(c.entries, k)
			continue
		}
		if first || e.expires.Before(oldestAt) {
			oldest, oldestAt, first = k, e.expires, false
		}
	}
	if len(c.entries) >= c.maxEntries && !first {
		delete(c.entries, oldest)
	}
}
