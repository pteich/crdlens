package k8s

import (
	"sync"
	"time"
)

// CacheEntry represents a cached item with an expiration
type CacheEntry struct {
	Value      interface{}
	Expiration int64
}

// Cache handles in-memory caching of K8s data
type Cache struct {
	mu    sync.RWMutex
	items map[string]CacheEntry
	ttl   time.Duration
}

// NewCache creates a new Cache with specified TTL
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		items: make(map[string]CacheEntry),
		ttl:   ttl,
	}
}

// Set adds an item to the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(c.ttl).UnixNano(),
	}
}

// Get retrieves an item from the cache if it exists and is not expired
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}

	if time.Now().UnixNano() > item.Expiration {
		return nil, false
	}

	return item.Value, true
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]CacheEntry)
}
