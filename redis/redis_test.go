package redis

import (
	"bytes"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
)

func TestRedisCache(t *testing.T) {
	pool := &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 60 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", "127.0.0.1:6379")
			if err != nil {
				return nil, err
			}
			return c, err
		},
	}

	conn := pool.Get()
	_, err := conn.Do("FLUSHALL")
	conn.Close()
	if err != nil {
		// TODO: rather than skip the test, fall back to a faked redis server
		t.Skipf("skipping test; no server running at localhost:6379")
	}

	cache := NewWithClient(pool)

	key := "testKey"
	_, ok := cache.Get(key)
	if ok {
		t.Fatal("retrieved key before adding it")
	}

	val := []byte("some bytes")
	cache.Set(key, val, 5*time.Second)

	retVal, ok := cache.Get(key)
	if !ok {
		t.Fatal("could not retrieve an element we just added")
	}
	if !bytes.Equal(retVal, val) {
		t.Fatal("retrieved a different value than what we put in")
	}

	cache.Delete(key)

	_, ok = cache.Get(key)
	if ok {
		t.Fatal("deleted key still present")
	}

	cache.Set(key, val, 0*time.Second)
	time.Sleep(500 * time.Millisecond)
	retVal, ok = cache.Get(key)
	if ok {
		t.Fatal("retrieved an element that should have expired")
	}
}

func TestConcurrency(t *testing.T) {
	type testData struct {
		key      string
		value    []byte
		duration time.Duration
	}

	data := []testData{
		testData{"one", []byte("1"), time.Second * 1},
		testData{"two", []byte("2"), time.Second * 1},
		testData{"three", []byte("3"), time.Second * 1},
		testData{"four", []byte("4"), time.Second * 1},
		testData{"five", []byte("5"), time.Second * 1},
		testData{"six", []byte("6"), time.Second * 1},
		testData{"seven", []byte("7"), time.Second * 1},
		testData{"eight", []byte("8"), time.Second * 1},
		testData{"nine", []byte("9"), time.Second * 1},
		testData{"ten", []byte("10"), time.Second * 1},
	}

	pool := &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 60 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", "127.0.0.1:6379")
			if err != nil {
				return nil, err
			}
			return c, err
		},
	}

	cache := NewWithClient(pool)

	retChan := make(chan error)

	// Add the entries
	for _, d := range data {
		go func(c *Cache, ret chan error, d testData) {
			c.Set(d.key, d.value, d.duration)
			retChan <- nil
		}(cache, retChan, d)
	}

	// Wait for the goroutines to finish
	for i := 0; i < len(data); i++ {
		<-retChan
	}

	// Check the entries exist
	for _, d := range data {
		go func(c *Cache, ret chan error, d testData) {
			retVal, ok := c.Get(d.key)
			if !ok {
				t.Log("could not retrieve an element we just added")
				t.Fail()
			}
			if !bytes.Equal(retVal, d.value) {
				t.Log("retrieved a different value than what we put in")
				t.Fail()
			}
			retChan <- nil
		}(cache, retChan, d)
	}

	// Wait for the goroutines to finish
	for i := 0; i < len(data); i++ {
		<-retChan
	}

	// Wait for the strings to expire
	time.Sleep(time.Second * 2)

	// Make sure we get invalid responses since they were expired by redis
	for _, d := range data {
		go func(c *Cache, ret chan error, d testData) {
			_, ok := c.Get(d.key)
			if ok {
				t.Log("retrieved an element that should have expired")
				t.Fail()
			}
			retChan <- nil
		}(cache, retChan, d)
	}

	// Wait for the goroutines to finish
	for i := 0; i < len(data); i++ {
		<-retChan
	}

}
