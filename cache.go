package merge

import (
	"sync"
)

// SimpleCache is a simple in-memory cache that uses a sync.Map to store key-value pairs.
// It is safe for concurrent use, but it is grow-only.
type SimpleCache struct {
	mu    sync.Mutex
	items sync.Map
}

// NewCache creates a new SimpleCache.
func NewCache() *SimpleCache {
	return &SimpleCache{}
}

// Get returns the value for the given key, and a boolean indicating whether the key was found.
func (c *SimpleCache) Get(key string) (string, bool) {
	if value, ok := c.items.Load(key); ok {
		return value.(string), true
	}
	return "", false
}

// Set sets the value for the given key.
func (c *SimpleCache) Set(key string, data string) {
	c.items.Store(key, data)
}

// Clear removes all items from the cache.
func (c *SimpleCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = sync.Map{}
}
