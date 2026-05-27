package receptionist

import (
	"sync"
	"time"
)

const convCacheTTL = 6 * time.Hour

type convCacheEntry struct {
	lastSeen time.Time
	value    any
}

// convCache is a simple TTL map for per-conversation in-memory state.
type convCache struct {
	mu      sync.Mutex
	entries map[string]convCacheEntry
}

func newConvCache() *convCache {
	return &convCache{entries: make(map[string]convCacheEntry)}
}

func (c *convCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evictLocked(time.Now())
	e, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Since(e.lastSeen) > convCacheTTL {
		delete(c.entries, key)
		return nil, false
	}
	e.lastSeen = time.Now()
	c.entries[key] = e
	return e.value, true
}

func (c *convCache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = convCacheEntry{lastSeen: time.Now(), value: value}
}

// GetOrSet returns the existing value for key, or stores and returns newValue.
// This is atomic under the cache lock and prevents races on first initialization.
func (c *convCache) GetOrSet(key string, newValue func() any) any {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evictLocked(time.Now())
	if e, ok := c.entries[key]; ok && time.Since(e.lastSeen) <= convCacheTTL {
		e.lastSeen = time.Now()
		c.entries[key] = e
		return e.value
	}
	v := newValue()
	c.entries[key] = convCacheEntry{lastSeen: time.Now(), value: v}
	return v
}

func (c *convCache) evictLocked(now time.Time) {
	for k, e := range c.entries {
		if now.Sub(e.lastSeen) > convCacheTTL {
			delete(c.entries, k)
		}
	}
}
