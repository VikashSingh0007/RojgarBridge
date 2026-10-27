package cache

import (
	"sync"
	"time"
)

type item struct {
	value      interface{}
	expiration int64
}

// Cache provides a thread-safe in-memory key-value cache with TTL expiration
type Cache struct {
	mu    sync.RWMutex
	items map[string]item
}

func NewCache(cleanupInterval time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]item),
	}
	if cleanupInterval > 0 {
		go c.startCleanup(cleanupInterval)
	}
	return c
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	it, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		return nil, false
	}

	if it.expiration > 0 && time.Now().UnixNano() > it.expiration {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}

	return it.value, true
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()
	c.items[key] = item{
		value:      value,
		expiration: exp,
	}
	c.mu.Unlock()
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *Cache) DeletePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.items, k)
		}
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]item)
	c.mu.Unlock()
}

func (c *Cache) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		now := time.Now().UnixNano()
		c.mu.Lock()
		for k, it := range c.items {
			if it.expiration > 0 && now > it.expiration {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
