// Package diskcache provided an implementation of httpcache.Cache that uses the diskv package
// to supplement an in-memory map with persistent storage
//
package diskcache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/peterbourgon/diskv"
	"io"
)

// Cache is an implementation of httpcache.Cache that supplements the in-memory map with persistent storage
type Cache struct {
	d *diskv.Diskv
}

func (c *Cache) Get(key string) (resp []byte, ok bool) {
	fmt.Println(key)
	key = keyToFilename(key)
	fmt.Println("Get cache", key)
	resp, err := c.d.Read(key)
	if err != nil {
		return []byte{}, false
	}
	return resp, true
}

func (c *Cache) Set(key string, resp []byte) {
	key = keyToFilename(key)
	fmt.Println("Set cache ", key)
	c.d.WriteAndSync(key, resp)
}

func (c *Cache) Delete(key string) {
	key = keyToFilename(key)
	c.d.Erase(key)
}

func keyToFilename(key string) string {
	h := md5.New()
	io.WriteString(h, key)
	return hex.EncodeToString(h.Sum(nil))
}

// New returns a new Cache that will store files in basePath
func New(basePath string) *Cache {
	d := diskv.New(diskv.Options{BasePath: basePath})
	cache := &Cache{d: d}
	return cache
}
