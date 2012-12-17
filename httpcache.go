// Package httpcache provides a http.RoundTripper implementation that works as a 
// mostly RFC-compliant cache for http responses.
//
// It is only suitable for use as a 'private' cache (i.e. for a web-browser or an API-client
// and not for a shared proxy).
//
// 'max-stale' set on a request is not currently respected. (max-age and min-fresh both are.)
package httpcache

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"
)

var _ = fmt.Println

const (
	stale = iota
	fresh
	transparent
	// Header added to responses that are returned from the cache
	XFromCache = "X-From-Cache"
)

// A Cache interface is used by the Transport to store and retrieve responses.
type Cache interface {
	// Get returns the []byte representation of a cached response and a bool
	// set to true if the value isn't empty
	Get(key string) (responseBytes []byte, ok bool)
	// Set stores the []byte representation of a response against a key
	Set(key string, responseBytes []byte)
	// Delete removes the value associated with the key
	Delete(key string)
}

// MemoryCache is an implemtation of Cache that stores responses in an in-memory map.
type MemoryCache struct {
	sync.RWMutex
	items map[string][]byte
}

func (c *MemoryCache) Get(key string) (resp []byte, ok bool) {
	c.RLock()
	defer c.RUnlock()
	resp, ok = c.items[key]
	return resp, ok
}

func (c *MemoryCache) Set(key string, resp []byte) {
	c.Lock()
	defer c.Unlock()
	c.items[key] = resp
}

func (c *MemoryCache) Delete(key string) {
	c.Lock()
	defer c.Unlock()
	delete(c.items, key)
}

// NewMemoryCache returns a new Cache that will store items in an in-memory map
func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{items: map[string][]byte{}, RWMutex: sync.RWMutex{}}
	return c
}

// Transport is an implementation of http.RoundTripper that will return values from a cache
// where possible (avoiding a network request) and will additionally add validators (etag/if-modified-since)
// to repeated requests allowing servers to return 304 / Not Modified
//
// Note: this means that both the request and response are potentially modified
type Transport struct {
	// The RoundTripper interface actually used to make requests
	// If this follows redirects, then only the final response's cache-control will be taken into account
	transport http.RoundTripper
	cache     Cache
	// If true, responses returned from the cache will be given an extra header, X-From-Cache
	MarkCachedResponses bool
}

// NewTransport returns a new Transport using the default HTTP Transport and the
// provided Cache implementation, with MarkCachedResponses set to true
func NewTransport(c Cache) *Transport {
	t := &Transport{transport: http.DefaultTransport, cache: c, MarkCachedResponses: true}
	return t
}

// varyMatches will return false unless all of the cached values for the headers listed in Vary
// match the new request
func varyMatches(cachedResp *http.Response, req *http.Request) bool {
	respVarys := cachedResp.Header.Get("vary")
	for _, header := range strings.Split(respVarys, ",") {
		header = http.CanonicalHeaderKey(strings.Trim(header, " "))
		if header != "" && req.Header.Get(header) != cachedResp.Header.Get("X-Varied-"+header) {
			return false
		}
	}
	return true
}

// RoundTrip takes a Request and returns a Response
//
// If there is a fresh Response already in cache, then it will be returned without connecting to
// the server.
//
// If there is a stale Response, then any validators it contains will be set on the new request
// to give the server a chance to respond with NotModified. If this happens, then the cached Response
// will be returned.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	cacheKey := req.URL.String()
	cachedVal, ok := t.cache.Get(cacheKey)
	cacheableMethod := req.Method == "GET" || req.Method == "HEAD"
	if !cacheableMethod {
		// Need to invalidate an existing value
		t.cache.Delete(cacheKey)
	}
	if ok && cacheableMethod && req.Header.Get("range") == "" {
		cachedResp, err := responseFromCache(cachedVal, req)
		if err == nil {
			if t.MarkCachedResponses {
				cachedResp.Header.Set(XFromCache, "1")
			}

			if varyMatches(cachedResp, req) {
				// Can only use cached value if the new request doesn't Vary significantly
				freshness := getfreshness(cachedResp.Header, req.Header)
				if freshness == fresh {
					return cachedResp, nil
				}

				if freshness == stale {
					// Add validators if caller hasn't already done so
					etag := cachedResp.Header.Get("etag")
					if etag != "" && req.Header.Get("etag") == "" {
						req.Header.Set("if-none-match", etag)
					}
					lastModified := cachedResp.Header.Get("last-modified")
					if lastModified != "" && req.Header.Get("last-modified") == "" {
						req.Header.Set("if-modified-since", lastModified)
					}
				}
			}

			resp, err = t.transport.RoundTrip(req)
			if err == nil && req.Method == "GET" && resp.StatusCode == http.StatusNotModified {
				// Replace the 304 response with the one from cache, but update with some new headers
				headersToMerge := getHopByHopHeaders(resp)
				for _, headerKey := range headersToMerge {
					cachedResp.Header.Set(headerKey, resp.Header.Get(headerKey))
				}
				cachedResp.Status = http.StatusText(http.StatusOK)
				cachedResp.StatusCode = http.StatusOK

				resp = cachedResp
			} else {
				if err != nil || resp.StatusCode != http.StatusOK {
					t.cache.Delete(cacheKey)
				}
			}
		}
	} else {
		reqCacheControl := parseCacheControl(req.Header)
		if _, ok := reqCacheControl["only-if-cached"]; ok {
			resp = newGatewayTimeoutResponse(req)
		} else {
			resp, err = t.transport.RoundTrip(req)
		}
	}
	reqCacheControl := parseCacheControl(req.Header)
	respCacheControl := parseCacheControl(resp.Header)

	if canStore(reqCacheControl, respCacheControl) {
		vary := resp.Header.Get("Vary")
		for _, varyKey := range strings.Split(vary, ",") {
			varyKey = http.CanonicalHeaderKey(strings.Trim(varyKey, " "))
			fakeHeader := "X-Varied-" + varyKey
			reqValue := req.Header.Get(varyKey)
			if reqValue != "" {
				resp.Header.Set(fakeHeader, reqValue)
			}
		}
		respBytes, err := httputil.DumpResponse(resp, true)
		if err == nil {
			// fmt.Println("Set cache", string(respBytes))
			t.cache.Set(cacheKey, respBytes)
		}
	} else {
		t.cache.Delete(cacheKey)
	}
	return resp, nil
}

// getfreshness will return one of fresh/stale/transparent based on the cache-control
// values of the request and the response
// 
// fresh indicates the response can be returned
// stale indicates that the response needs validating before it is returned
// transparent indicates the response should not be used to fulfil the request
//
// Because this is only a private cache, 'public' and 'private' in cache-control aren't
// signficant. Similarly, smax-age isn't used.
//
// Limitation: max-stale is not taken into account. It should be.
func getfreshness(respHeaders, reqHeaders http.Header) (freshness int) {
	respCacheControl := parseCacheControl(respHeaders)
	reqCacheControl := parseCacheControl(reqHeaders)
	if _, ok := reqCacheControl["no-cache"]; ok {
		return transparent
	}
	if _, ok := respCacheControl["no-cache"]; ok {
		return stale
	}
	if _, ok := reqCacheControl["only-if-cached"]; ok {
		return fresh
	}
	dateHeader := respHeaders.Get("date")
	if dateHeader != "" {
		date, err := time.Parse(time.RFC1123, dateHeader)
		if err != nil {
			return stale
		}
		currentAge := time.Since(date)
		var lifetime time.Duration
		zeroDuration, _ := time.ParseDuration("0s")
		// If a response includes both an Expires header and a max-age directive, 
		// the max-age directive overrides the Expires header, even if the Expires header is more restrictive.
		if maxAge, ok := respCacheControl["max-age"]; ok {
			lifetime, err = time.ParseDuration(maxAge + "s")
			if err != nil {
				lifetime = zeroDuration
			}
		} else {
			if expiresHeader, ok := respCacheControl["expires"]; ok {
				expires, err := time.Parse(time.RFC1123, expiresHeader)
				if err != nil {
					lifetime = zeroDuration
				} else {
					lifetime = expires.Sub(date)
				}
			}
		}

		if maxAge, ok := reqCacheControl["max-age"]; ok {
			// the client is willing to accept a response whose age is no greater than the specified time in seconds
			lifetime, err = time.ParseDuration(maxAge + "s")
			if err != nil {
				lifetime = zeroDuration
			}
		}
		if minfresh, ok := reqCacheControl["min-fresh"]; ok {
			//  the client wants a response that will still be fresh for at least the specified number of seconds.
			minfreshDuration, err := time.ParseDuration(minfresh + "s")
			if err != nil {
				currentAge = time.Duration(currentAge.Nanoseconds() + minfreshDuration.Nanoseconds())
			}
		}

		if lifetime > currentAge {
			return fresh
		}

	}
	return stale
}

func getHopByHopHeaders(resp *http.Response) []string {
	// These headers are always hop-by-hop
	headers := []string{"connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailers", "transfer-encoding", "upgrade"}

	for _, extra := range strings.Split(resp.Header.Get("connection"), ",") {
		// any header listed in connection, if present, is also considered hop-by-hop
		if strings.Trim(extra, " ") != "" {
			headers = append(headers, extra)
		}
	}
	return headers
}

func canStore(reqCacheControl, respCacheControl cacheControl) (canStore bool) {
	if _, ok := respCacheControl["no-store"]; ok {
		return false
	}
	if _, ok := reqCacheControl["no-store"]; ok {
		return false
	}
	return true
}

func responseFromCache(cachedVal []byte, req *http.Request) (*http.Response, error) {
	b := bytes.NewBuffer(cachedVal)
	resp, err := http.ReadResponse(bufio.NewReader(b), req)
	return resp, err
}

func newGatewayTimeoutResponse(req *http.Request) *http.Response {
	var braw bytes.Buffer
	braw.WriteString("HTTP/1.1 504 Gateway Timeout\r\n\r\n")
	resp, err := http.ReadResponse(bufio.NewReader(&braw), req)
	if err != nil {
		panic(err)
	}
	return resp
}

type cacheControl map[string]string

func parseCacheControl(headers http.Header) cacheControl {
	cc := cacheControl{}
	ccHeader := headers.Get("Cache-Control")
	for _, part := range strings.Split(ccHeader, ",") {
		part = strings.Trim(part, " ")
		if part == "" {
			continue
		}
		if strings.ContainsRune(part, '=') {
			keyval := strings.Split(part, "=")
			cc[strings.Trim(keyval[0], " ")] = strings.Trim(keyval[1], ",")
		} else {
			cc[part] = "1"
		}
	}
	return cc
}

// NewMemoryCacheTransport returns a new Transport using the in-memory cache implementation
func NewMemoryCacheTransport() *Transport {
	c := NewMemoryCache()
	t := NewTransport(c)
	return t
}
