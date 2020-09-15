package test_test

import (
	"testing"

	"github.com/lggomez/httpcache/v2"
	"github.com/lggomez/httpcache/v2/test"
)

func TestMemoryCache(t *testing.T) {
	test.Cache(t, httpcache.NewMemoryCache())
}
