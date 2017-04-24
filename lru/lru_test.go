package lru

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/gregjones/httpcache"
)

func TestLRU(t *testing.T) {
	cache := httpcache.NewMemoryCache()
	lru := New(cache, 10)
	tests := []struct {
		key     string
		val     []byte
		present []string
		absent  []string
	}{
		{"key1", randBytes(4), []string{"key1"}, []string{"unknowkey"}},                // cap: 6
		{"key2", randBytes(4), []string{"key2", "key1"}, []string{}},                   // cap: 2
		{"key3", randBytes(4), []string{"key3", "key2"}, []string{"key1"}},             // cap: 2
		{"key4", randBytes(6), []string{"key4", "key3"}, []string{"key2"}},             // cap: 0
		{"key5", randBytes(12), []string{"key5"}, []string{"key4", "key3"}},            // cap: -2
		{"key6", randBytes(1), []string{"key6"}, []string{"key5"}},                     // cap: 9
		{"key7", randBytes(1), []string{"key7", "key6"}, []string{}},                   // cap: 8
		{"key8", randBytes(8), []string{"key8", "key7", "key6"}, []string{}},           // cap: 0
		{"key7", randBytes(1), []string{"key7", "key8", "key6"}, []string{}},           // cap: 0
		{"key9", randBytes(1), []string{"key9", "key7", "key8"}, []string{"key6"}},     // cap: 0
		{"key8", randBytes(9), []string{"key8", "key9"}, []string{"key7"}},             // cap: 0
		{"key10", randBytes(1), []string{"key10", "key8"}, []string{"key9"}},           // cap: 0
		{"key8", randBytes(6), []string{"key8", "key10"}, []string{}},                  // cap: 3
		{"key11", randBytes(3), []string{"key11", "key8", "key10"}, []string{}},        // cap: 0
		{"key12", randBytes(5), []string{"key12", "key11"}, []string{"key8", "key10"}}, // cap: 2
	}

	for _, test := range tests {
		lru.Set(test.key, test.val)

		for _, key := range test.present {
			if val, exists := cache.Get(key); !exists {
				t.Errorf("expected '%s' to be in the cache after inserting '%s'", key, test.key)
			} else if test.key == key && bytes.Compare(test.val, val) != 0 {
				t.Errorf("value mismatch for '%s': got '%v', want '%v'", key, val, test.val)
			}
		}

		for _, key := range test.absent {
			if _, exists := cache.Get(key); exists {
				t.Errorf("unexpected item in cache '%s' after inserting '%s'", key, test.key)
			}
		}
	}
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}
