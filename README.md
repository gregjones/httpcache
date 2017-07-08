httpcache
=========

[![Build Status](https://travis-ci.org/gregjones/httpcache.svg?branch=master)](https://travis-ci.org/gregjones/httpcache) [![GoDoc](https://godoc.org/github.com/gregjones/httpcache?status.svg)](https://godoc.org/github.com/gregjones/httpcache)

Package httpcache provides a http.RoundTripper implementation that works as a mostly RFC-compliant cache for http responses.

It is only suitable for use as a 'private' cache (i.e. for a web-browser or an API-client and not for a shared proxy).

Cache Backends
--------------

- The built-in 'memory' cache stores responses in an in-memory map.
- [`github.com/gregjones/httpcache/diskcache`](https://github.com/gregjones/httpcache/tree/master/diskcache) provides a filesystem-backed cache using the [diskv](https://github.com/peterbourgon/diskv) library.
- [`github.com/gregjones/httpcache/memcache`](https://github.com/gregjones/httpcache/tree/master/memcache) provides memcache implementations, for both App Engine and 'normal' memcache servers.
- [`sourcegraph.com/sourcegraph/s3cache`](https://sourcegraph.com/github.com/sourcegraph/s3cache) uses Amazon S3 for storage.
- [`github.com/gregjones/httpcache/leveldbcache`](https://github.com/gregjones/httpcache/tree/master/leveldbcache) provides a filesystem-backed cache using [leveldb](https://github.com/syndtr/goleveldb/leveldb).
- [`github.com/die-net/lrucache`](https://github.com/die-net/lrucache) provides an in-memory cache that will evict least-recently used entries.
- [`github.com/die-net/lrucache/twotier`](https://github.com/die-net/lrucache/tree/master/twotier) allows caches to be combined, for example to use lrucache above with a persistent disk-cache.

## Getting started

Below is a basic example of usage.
```
func httpCacheExample() {
    numOfRequests := 0
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Cache-Control", fmt.Sprintf("private, max-age=10"))
        if numOfRequests == 0 {
            w.Write([]byte("Hello!"))
        } else {
            w.Write([]byte("Goodbye!"))
        }
        numOfRequests++
    }))

    httpClient := &http.Client{
        Transport: httpcache.NewMemoryCacheTransport(),
    }
    makeRequest(ts, httpClient) // "Hello!"

    // The second request is under max-age, so the cache is used rather than hitting the server
    makeRequest(ts, httpClient) // "Hello!"

    // Sleep so the max-age is passed
    time.Sleep(time.Second * 11)

    makeRequest(ts, httpClient) // "Goodbye!"
}

func makeRequest(ts *httptest.Server, httpClient *http.Client) {
    resp, _ := httpClient.Get(ts.URL)
    var buf bytes.Buffer
    io.Copy(&buf, resp.Body)
    println(buf.String())
}
```

License
-------

-	[MIT License](LICENSE.txt)
