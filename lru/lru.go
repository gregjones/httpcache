// Package lru provides a lru cache algorithm over an existing cache.
package lru

import (
	"container/list"
	"sync"

	"github.com/gregjones/httpcache"
)

// Cache is an LRU cache. It is safe for concurrent access.
// It itself uses a cache for its underlying storage.
type Cache struct {
	c     httpcache.Cache
	mu    sync.Mutex
	cap   int
	items map[string]*cacheItem
	list  *list.List
}

type cacheItem struct {
	key     string
	size    int
	element *list.Element
}

// Get looks up a key's value from the cache and refreshes it.
func (c *Cache) Get(key string) (resp []byte, ok bool) {
	c.mu.Lock()
	item, ok := c.items[key]
	if !ok {
		c.mu.Unlock()
		return
	}
	c.list.MoveToFront(item.element)
	c.mu.Unlock()
	return c.c.Get(key)
}

// Set adds or refreshes a value in the cache.
func (c *Cache) Set(key string, resp []byte) {
	victims := []string{} // to prevent lock contention of slow storage
	var added int

	c.mu.Lock()
	if item, exists := c.items[key]; exists {
		c.list.MoveToFront(item.element)
		added = len(resp) - item.size
		item.size = len(resp)
	} else {
		item := &cacheItem{key: key, size: len(resp)}
		item.element = c.list.PushFront(item)
		c.items[key] = item
		added = item.size
	}
	c.cap -= added
	for c.cap < 0 && c.list.Len() > 1 {
		tail := c.list.Back()
		item := tail.Value.(*cacheItem)
		victims = append(victims, item.key)
		c.purge(item)
	}
	c.mu.Unlock()

	for _, key := range victims {
		c.c.Delete(key)
	}
	c.c.Set(key, resp)
}

// Delete removes the provided key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	if item, exists := c.items[key]; exists {
		c.purge(item)
	}
	c.mu.Unlock()

	c.c.Delete(key)
}

func (c *Cache) purge(item *cacheItem) {
	delete(c.items, item.key)
	c.list.Remove(item.element)
	c.cap += item.size
}

// New creates a new Cache with c as its underlying storage
// and a capacity of cap bytes.
func New(c httpcache.Cache, cap int) httpcache.Cache {
	return &Cache{
		c:     c,
		cap:   cap,
		items: make(map[string]*cacheItem),
		list:  list.New(),
	}
}
