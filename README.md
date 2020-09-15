httpcache
=========

![Go](https://github.com/lggomez/httpcache/v2/workflows/Go/badge.svg?branch=master)

Package httpcache provides a http.RoundTripper wrapper implementation that works as a mostly [RFC 7234](https://tools.ietf.org/html/rfc7234) compliant cached client for http responses.

It is only suitable for use as a 'private' cache (i.e. for a web-browser or an API-client and not for a shared proxy).

### Parent repo state
This project isn't actively maintained, per author:
>it works for what I, and seemingly others, want to do with it, and I consider it "done". That said, if you find any issues, please open a Pull Request and I will >try to review it. Any changes now that change the public API won't be considered.

### Fork change notes

This fork implements the following changes:
* All backend implementations are omitted in this package, thus deleted
* Added a debug mode to diagnose cache behavior and invalidations
* Changed the API to the following: `NewCachedClient(c Cache, client *http.Client, markCached bool, debug bool) Doer` This allows to treat the cache client instance as a wrapped http.Client implementation and use it accordingly

License
-------

-	[MIT License](LICENSE.txt)
