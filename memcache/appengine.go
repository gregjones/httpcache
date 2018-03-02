// +build appengine

// Package memcache provides an implementation of httpcache.Cache that uses App
// Engine's memcache package to store cached responses.
//
// When not built for Google App Engine, this package will provide an
// implementation that connects to a specified memcached server.  See the
// memcache.go file in this package for details.
package memcache

import (
	"crypto/md5"
	"fmt"

	"appengine"
	"appengine/memcache"
)

// Cache is an implementation of httpcache.Cache that caches responses in App
// Engine's memcache.
type Cache struct {
	appengine.Context
}

// cacheKey modifies an httpcache key for use in memcache.  Specifically, it
// prefixes keys to avoid collision with other data stored in memcache. It
// also uses the MD5 hash of the key in order to avoid exceeding the 250
// character length limit for a memcache key.
func cacheKey(key string) string {
	md5 := md5.Sum([]byte(key))

	return "httpcache:" + fmt.Sprintf("%x", md5)
}

// Get returns the response corresponding to key if present.
func (c *Cache) Get(key string) (resp []byte, ok bool) {
	item, err := memcache.Get(c.Context, cacheKey(key))
	if err != nil {
		if err != memcache.ErrCacheMiss {
			c.Context.Errorf("error getting cached response: %v", err)
		}
		return nil, false
	}
	return item.Value, true
}

// Set saves a response to the cache as key.
func (c *Cache) Set(key string, resp []byte) {
	item := &memcache.Item{
		Key:   cacheKey(key),
		Value: resp,
	}
	if err := memcache.Set(c.Context, item); err != nil {
		c.Context.Errorf("error caching response: %v", err)
	}
}

// Delete removes the response with key from the cache.
func (c *Cache) Delete(key string) {
	if err := memcache.Delete(c.Context, cacheKey(key)); err != nil {
		c.Context.Errorf("error deleting cached response: %v", err)
	}
}

// New returns a new Cache for the given context.
func New(ctx appengine.Context) *Cache {
	return &Cache{ctx}
}
