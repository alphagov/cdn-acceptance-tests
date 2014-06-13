package main

import (
	"net/http"
	"testing"
)

// Should cache first response and return it on second request without
// hitting origin again.
func TestCacheFirstResponse(t *testing.T) {
	testThreeRequestsAreCached(t, nil)
}

// Should cache responses with default TTL if the response doesn't specify
// a period itself.
func TestCacheDefaultTTL(t *testing.T) {
	t.Error("Not implemented")
}

// Should cache responses for the period defined in a `Expires: n` response
// header.
func TestCacheExpires(t *testing.T) {
	t.Error("Not implemented")
}

// Should cache responses for the period defined in a `Cache-Control:
// max-age=n` response header.
func TestCacheCacheControlMaxAge(t *testing.T) {
	t.Error("Not implemented")
}

// Should cache responses with a `Cache-Control: no-cache` header. Varnish
// doesn't respect this by default.
func TestCacheCacheControlNoCache(t *testing.T) {
	handler := func(w http.ResponseWriter) {
		w.Header().Set("Cache-Control", "no-cache")
	}

	testThreeRequestsAreCached(t, handler)
}

// Should cache responses with a status code of 404. It's a common
// misconception that 404 responses shouldn't be cached; they should because
// they can be expensive to generate.
func TestCache404Response(t *testing.T) {
	handler := func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusNotFound)
	}

	testThreeRequestsAreCached(t, handler)
}
