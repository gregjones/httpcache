httpcache
=========

A Transport for Go's http.Client that will cache responses according to the HTTP RFC

Package httpcache provides a http.RoundTripper implementation that works as a mostly RFC-compliant cache for http responses.

It is only suitable for use as a 'private' cache (i.e. for a web-browser or an API-client and not for a shared proxy).

**Documentation:** http://godoc.org/github.com/gregjones/httpcache

**License:** MIT (see LICENSE.txt)
