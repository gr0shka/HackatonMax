package places

import (
	"container/list"
	"sync"
	"time"

	"HackatonMax/internal/entity"
)

type cacheEntry struct {
	key       string
	value     []entity.Place
	expiresAt time.Time
}

// MemoryLRUTTLCache provides a thread-safe LRU cache with TTL eviction.
type MemoryLRUTTLCache struct {
	mu       sync.Mutex
	capacity int
	ttl      time.Duration
	items    map[string]*list.Element
	evict    *list.List
}

// NewMemoryLRUTTLCache creates a new cache with specified capacity and TTL.
func NewMemoryLRUTTLCache(capacity int, ttl time.Duration) *MemoryLRUTTLCache {
	if capacity <= 0 {
		capacity = 500
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &MemoryLRUTTLCache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*list.Element, capacity),
		evict:    list.New(),
	}
}

// Get retrieves cached places by key if present and not expired.
func (c *MemoryLRUTTLCache) Get(key string) ([]entity.Place, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.items[key]
	if !exists {
		return nil, false
	}

	entry := elem.Value.(*cacheEntry)
	if time.Now().After(entry.expiresAt) {
		c.evict.Remove(elem)
		delete(c.items, key)
		return nil, false
	}

	c.evict.MoveToFront(elem)
	result := make([]entity.Place, len(entry.value))
	copy(result, entry.value)
	return result, true
}

// Set stores places under the specified key with current TTL.
func (c *MemoryLRUTTLCache) Set(key string, value []entity.Place) {
	c.mu.Lock()
	defer c.mu.Unlock()

	copied := make([]entity.Place, len(value))
	copy(copied, value)

	if elem, exists := c.items[key]; exists {
		c.evict.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		entry.value = copied
		entry.expiresAt = time.Now().Add(c.ttl)
		return
	}

	if c.evict.Len() >= c.capacity {
		back := c.evict.Back()
		if back != nil {
			c.evict.Remove(back)
			oldEntry := back.Value.(*cacheEntry)
			delete(c.items, oldEntry.key)
		}
	}

	entry := &cacheEntry{
		key:       key,
		value:     copied,
		expiresAt: time.Now().Add(c.ttl),
	}
	elem := c.evict.PushFront(entry)
	c.items[key] = elem
}

// Len returns the current number of cached entries.
func (c *MemoryLRUTTLCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.evict.Len()
}
