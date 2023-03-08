package test_test

import (
	"testing"

	"github.com/secure-cloud-stack/httpcache"
	"github.com/secure-cloud-stack/httpcache/test"
)

func TestMemoryCache(t *testing.T) {
	test.Cache(t, httpcache.NewMemoryCache())
}
