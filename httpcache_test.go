package httpcache

import (
	"fmt"
	. "launchpad.net/gocheck"
	"net"
	"net/http"
	"testing"
	"time"
)

var _ = fmt.Print

func Test(t *testing.T) { TestingT(t) }

type S struct {
	listener  net.Listener
	client    http.Client
	transport *Transport
}

var _ = Suite(&S{})

func (s *S) SetUpSuite(c *C) {
	t := NewMemoryCacheTransport()
	client := http.Client{Transport: t}
	s.transport = t
	s.client = client

	ln, err := net.Listen("tcp", ":9090")
	if err != nil {
		panic(err)
	}
	s.listener = ln

	http.HandleFunc("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=3600")
	}))

	http.HandleFunc("/nostore", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
	}))

	http.HandleFunc("/etag", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		etag := "124567"
		if r.Header.Get("if-none-match") == etag {
			w.WriteHeader(http.StatusNotModified)
		}
		w.Header().Set("etag", etag)
	}))

	http.HandleFunc("/lastmodified", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lm := "Fri, 14 Dec 2010 01:01:50 GMT"
		if r.Header.Get("if-modified-since") == lm {
			w.WriteHeader(http.StatusNotModified)
		}
		w.Header().Set("last-modified", lm)
	}))

	http.HandleFunc("/varyaccept", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Vary", "Accept")
		w.Write([]byte("Some text content"))
	}))

	http.HandleFunc("/doublevary", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Vary", "Accept, Accept-Language")
		w.Write([]byte("Some text content"))
	}))
	http.HandleFunc("/varyunused", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Vary", "X-Madeup-Header")
		w.Write([]byte("Some text content"))
	}))

	go http.Serve(s.listener, nil)
}

func (s *S) TearDownSuite(c *C) {
	err := s.listener.Close()
	if err != nil {
		panic(err)
	}
}

func (s *S) TearDownTest(c *C) {
	s.transport.Cache = NewMemoryCache()
}

func (s *S) TestGetOnlyIfCachedHit(c *C) {
	req, err := http.NewRequest("GET", "http://localhost:9090/", nil)
	c.Assert(err, IsNil)
	resp, err := s.client.Do(req)
	defer resp.Body.Close()
	c.Assert(resp.Header.Get(XFromCache), Equals, "")

	req2, err2 := http.NewRequest("GET", "http://localhost:9090/", nil)
	req2.Header.Add("cache-control", "only-if-cached")
	resp2, err2 := s.client.Do(req)
	defer resp2.Body.Close()
	c.Assert(err2, IsNil)
	c.Assert(resp2.Header.Get(XFromCache), Equals, "1")
	c.Assert(resp2.StatusCode, Equals, 200)
}

func (s *S) TestGetOnlyIfCachedMiss(c *C) {
	req, err := http.NewRequest("GET", "http://localhost:9090/", nil)
	req.Header.Add("cache-control", "only-if-cached")
	resp, err := s.client.Do(req)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	c.Assert(resp.Header.Get(XFromCache), Equals, "")
	c.Assert(resp.StatusCode, Equals, 504)
}

func (s *S) TestGetNoStoreRequest(c *C) {
	req, err := http.NewRequest("GET", "http://localhost:9090/", nil)
	req.Header.Add("Cache-Control", "no-store")
	resp, err := s.client.Do(req)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	c.Assert(resp.Header.Get(XFromCache), Equals, "")

	resp2, err2 := s.client.Do(req)
	defer resp2.Body.Close()
	c.Assert(err2, IsNil)
	c.Assert(resp2.Header.Get(XFromCache), Equals, "")
}

func (s *S) TestGetNoStoreResponse(c *C) {
	req, err := http.NewRequest("GET", "http://localhost:9090/nostore", nil)
	resp, err := s.client.Do(req)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	c.Assert(resp.Header.Get(XFromCache), Equals, "")

	resp2, err2 := s.client.Do(req)
	defer resp2.Body.Close()
	c.Assert(err2, IsNil)
	c.Assert(resp2.Header.Get(XFromCache), Equals, "")
}

func (s *S) TestGetWithEtag(c *C) {
	req, err := http.NewRequest("GET", "http://localhost:9090/etag", nil)
	resp, err := s.client.Do(req)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	c.Assert(resp.Header.Get(XFromCache), Equals, "")

	resp2, err2 := s.client.Do(req)
	defer resp2.Body.Close()
	c.Assert(err2, IsNil)
	c.Assert(resp2.Header.Get(XFromCache), Equals, "1")
}

func (s *S) TestGetWithLastModified(c *C) {
	req, err := http.NewRequest("GET", "http://localhost:9090/lastmodified", nil)
	resp, err := s.client.Do(req)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	c.Assert(resp.Header.Get(XFromCache), Equals, "")

	resp2, err2 := s.client.Do(req)
	defer resp2.Body.Close()
	c.Assert(err2, IsNil)
	c.Assert(resp2.Header.Get(XFromCache), Equals, "1")
}

func (s *S) TestGetWithVary(c *C) {
	req, err := http.NewRequest("GET", "http://localhost:9090/varyaccept", nil)
	req.Header.Set("Accept", "text/plain")
	resp, err := s.client.Do(req)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	c.Assert(resp.Header.Get("Vary"), Equals, "Accept")

	resp2, err2 := s.client.Do(req)
	defer resp2.Body.Close()
	c.Assert(err2, IsNil)
	c.Assert(resp2.Header.Get(XFromCache), Equals, "1")

	req.Header.Set("Accept", "text/html")
	resp3, err3 := s.client.Do(req)
	defer resp3.Body.Close()
	c.Assert(err3, IsNil)
	c.Assert(resp3.Header.Get(XFromCache), Equals, "")

	req.Header.Set("Accept", "")
	resp4, err4 := s.client.Do(req)
	defer resp4.Body.Close()
	c.Assert(err4, IsNil)
	c.Assert(resp4.Header.Get(XFromCache), Equals, "")
}

func (s *S) TestGetWithDoubleVary(c *C) {
	req, err := http.NewRequest("GET", "http://localhost:9090/doublevary", nil)
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("Accept-Language", "da, en-gb;q=0.8, en;q=0.7")
	resp, err := s.client.Do(req)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	c.Assert(resp.Header.Get("Vary"), Not(Equals), "")

	resp2, err2 := s.client.Do(req)
	defer resp2.Body.Close()
	c.Assert(err2, IsNil)
	c.Assert(resp2.Header.Get(XFromCache), Equals, "1")

	req.Header.Set("Accept-Language", "")
	resp3, err3 := s.client.Do(req)
	defer resp3.Body.Close()
	c.Assert(err3, IsNil)
	c.Assert(resp3.Header.Get(XFromCache), Equals, "")

	req.Header.Set("Accept-Language", "da")
	resp4, err4 := s.client.Do(req)
	defer resp4.Body.Close()
	c.Assert(err4, IsNil)
	c.Assert(resp4.Header.Get(XFromCache), Equals, "")
}

func (s *S) TestGetVaryUnused(c *C) {
	req, err := http.NewRequest("GET", "http://localhost:9090/varyunused", nil)
	req.Header.Set("Accept", "text/plain")
	resp, err := s.client.Do(req)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	c.Assert(resp.Header.Get("Vary"), Not(Equals), "")

	resp2, err2 := s.client.Do(req)
	defer resp2.Body.Close()
	c.Assert(err2, IsNil)
	c.Assert(resp2.Header.Get(XFromCache), Equals, "1")
}

func (s *S) TestParseCacheControl(c *C) {
	h := http.Header{}
	for _ = range parseCacheControl(h) {
		c.Fatal("cacheControl should be empty")
	}

	h.Set("cache-control", "no-cache")
	cc := parseCacheControl(h)
	c.Assert(cc["no-cache"], Equals, "1")

	h.Set("cache-control", "no-cache, max-age=3600")
	cc = parseCacheControl(h)
	c.Assert(cc["no-cache"], Equals, "1")
	c.Assert(cc["max-age"], Equals, "3600")
}

func (s *S) TestNoCacheRequestExpiration(c *C) {
	respHeaders := http.Header{}
	respHeaders.Set("Cache-Control", "max-age=7200")
	reqHeaders := http.Header{}
	reqHeaders.Set("Cache-Control", "no-cache")

	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, transparent)
}

func (s *S) TestNoCacheResponseExpiration(c *C) {
	respHeaders := http.Header{}
	respHeaders.Set("Cache-Control", "no-cache")
	respHeaders.Set("Expires", "Wed, 19 Apr 3000 11:43:00 GMT")
	reqHeaders := http.Header{}

	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, stale)
}

func (s *S) TestReqMustRevalidate(c *C) {
	// not paying attention to request setting max-stale means never returning stale
	// responses, so always acting as if must-revalidate is set
	respHeaders := http.Header{}
	reqHeaders := http.Header{}
	reqHeaders.Set("Cache-Control", "must-revalidate")

	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, stale)
}

func (s *S) TestRespMustRevalidate(c *C) {
	respHeaders := http.Header{}
	respHeaders.Set("Cache-Control", "must-revalidate")
	reqHeaders := http.Header{}

	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, stale)
}

func (s *S) TestFreshExpiration(c *C) {
	now := time.Now()
	respHeaders := http.Header{}
	respHeaders.Set("date", now.Format(time.RFC1123))
	respHeaders.Set("expires", now.Add(time.Duration(2)*time.Second).Format(time.RFC1123))

	reqHeaders := http.Header{}
	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, fresh)

	time.Sleep(3 * time.Second)
	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, stale)
}

func (s *S) TestMaxAge(c *C) {
	now := time.Now()
	respHeaders := http.Header{}
	respHeaders.Set("date", now.Format(time.RFC1123))
	respHeaders.Set("cache-control", "max-age=2")

	reqHeaders := http.Header{}
	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, fresh)

	time.Sleep(3 * time.Second)
	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, stale)
}

func (s *S) TestMaxAgeZero(c *C) {
	now := time.Now()
	respHeaders := http.Header{}
	respHeaders.Set("date", now.Format(time.RFC1123))
	respHeaders.Set("cache-control", "max-age=0")

	reqHeaders := http.Header{}
	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, stale)
}

func (s *S) TestBothMaxAge(c *C) {
	now := time.Now()
	respHeaders := http.Header{}
	respHeaders.Set("date", now.Format(time.RFC1123))
	respHeaders.Set("cache-control", "max-age=2")

	reqHeaders := http.Header{}
	reqHeaders.Set("cache-control", "max-age=0")
	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, stale)
}

func (s *S) TestMinFreshWithExpires(c *C) {
	now := time.Now()
	respHeaders := http.Header{}
	respHeaders.Set("date", now.Format(time.RFC1123))
	respHeaders.Set("expires", now.Add(time.Duration(2)*time.Second).Format(time.RFC1123))

	reqHeaders := http.Header{}
	reqHeaders.Set("cache-control", "min-fresh=1")
	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, fresh)

	reqHeaders = http.Header{}
	reqHeaders.Set("cache-control", "min-fresh=2")
	c.Assert(getFreshness(respHeaders, reqHeaders), Equals, stale)
}
