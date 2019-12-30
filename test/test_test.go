package test_test

import (
	"testing"

	"github.com/lggomez/httpcache"
	"github.com/lggomez/httpcache/test"
)

func TestMemoryCache(t *testing.T) {
	test.Cache(t, httpcache.NewMemoryCache())
}
