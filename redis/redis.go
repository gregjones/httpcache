// Package redis provides a redis interface for http caching.
package redis

import (
	"time"

	"github.com/garyburd/redigo/redis"
)

// cache is an implementation of httpcache.Cache that caches responses in a
// redis server.
type Cache struct {
	*redis.Pool
}

// cacheKey modifies an httpcache key for use in redis. Specifically, it
// prefixes keys to avoid collision with other data stored in redis.
func cacheKey(key string) string {
	return "rediscache:" + key
}

// Get returns the response corresponding to key if present.
func (c *Cache) Get(key string) (resp []byte, ok bool) {
	conn := c.Pool.Get()
	defer conn.Close()
	item, err := redis.Bytes(conn.Do("GET", cacheKey(key)))
	if err != nil {
		return nil, false
	}
	return item, true
}

// Set saves a response to the cache as key.
func (c *Cache) Set(key string, resp []byte, duration time.Duration) {
	conn := c.Pool.Get()
	conn.Do("SETEX", cacheKey(key), (int)(duration.Seconds()), resp)
	conn.Close()
}

// Delete removes the response with key from the cache.
func (c *Cache) Delete(key string) {
	conn := c.Pool.Get()
	conn.Do("DEL", cacheKey(key))
	conn.Close()
}

// NewWithClient returns a new Cache with the given redis pool.
func NewWithClient(client *redis.Pool) *Cache {
	return &Cache{client}
}
