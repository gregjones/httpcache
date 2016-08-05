package rediscache

import (
	"bytes"
	"testing"

	"github.com/soveran/redisurl"
)

const testServer = "redis://localhost:6379"

func TestRedisCache(t *testing.T) {
	conn, err := redisurl.ConnectToURL(testServer)
	if err != nil {
		// TODO: rather than skip the test, fall back to a faked redis server
		t.Skipf("skipping test; no server running at %s", testServer)
	}
	conn.Do("FLUSHALL")

	cache := NewWithClient(conn)

	key := "testKey"
	_, ok := cache.Get(key)
	if ok {
		t.Fatal("retrieved key before adding it")
	}

	val := []byte("some bytes")
	cache.Set(key, val)

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
}
