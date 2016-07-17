// Package boltdbcache provides an implementation of httpcache.Cache that
// uses github.com/boltdb/bolt
package boltdbcache

import (
	"github.com/boltdb/bolt"
)

// Cache is an implementation of httpcache.Cache with boltdb storage
type Cache struct {
	db *bolt.DB
	bucket []byte
}

// Get returns the response corresponding to key if present
func (c *Cache) Get(key string) (resp []byte, ok bool) {
	err := c.db.View(func(tx *bolt.Tx) error {
        resp = tx.Bucket(c.bucket).Get([]byte(key))
        return nil
    })
    if resp == nil || err != nil {
    	return nil, false
    }
	return resp, true
}

// Set saves a response to the cache as key
func (c *Cache) Set(key string, resp []byte) {
	c.db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(c.bucket)
        return bucket.Put([]byte(key), resp)
	});
}

// Delete removes the response with key from the cache
func (c *Cache) Delete(key string) {
	c.db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(c.bucket)
        return bucket.Delete([]byte(key))
	});
}

// New returns a new Cache that will store a boltdb in path using bucket (defaults to "cache")
func New(path string, bucket []byte) (*Cache, error) {
	c := &Cache{bucket: bucket}

	var err error
	c.db, err = bolt.Open(path, 0600, nil)
    if err != nil {
        return nil, err
    }

    err = c.db.Update(func(tx *bolt.Tx) error {
        _, err := tx.CreateBucketIfNotExists(c.bucket)
        return err
    })
    if err != nil {
        return nil, err
    }

	return c, nil
}

// NewWithDB returns a new Cache using the provided boltdb as underlying
// storage. Assumes the bucket is already created
func NewWithDB(db *bolt.DB, bucket []byte) *Cache {
	return &Cache{db, bucket}
}
