package main

import (
	"fmt"
	"net/http"
	"testing"
)

// Should send request to origin by default
func TestNoCacheNewRequestOrigin(t *testing.T) {
	ResetBackends(backendsByPriority)

	uuid := NewUUID()

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/%s", uuid) {
			w.Header().Set("EnsureOriginServed", uuid)
		}
	})

	sourceUrl := fmt.Sprintf("https://%s/%s", *edgeHost, uuid)

	req, _ := http.NewRequest("GET", sourceUrl, nil)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Status code expected 200, got %d", resp.StatusCode)
	}
	if d := resp.Header.Get("EnsureOriginServed"); d != uuid {
		t.Errorf("EnsureOriginServed header has not come from Origin: expected %q, got %q", uuid, d)
	}
}

// Should not cache the response to a POST request.
func TestNoCachePOST(t *testing.T) {
	ResetBackends(backendsByPriority)

	req := NewUniqueEdgeGET(t)
	req.Method = "POST"

	testThreeRequestsNotCached(t, req, nil)
}

// Should not cache the response to a request with a `Authorization` header.
func TestNoCacheHeaderAuthorization(t *testing.T) {
	ResetBackends(backendsByPriority)

	req := NewUniqueEdgeGET(t)
	req.Header.Set("Authorization", "Basic YXJlbnR5b3U6aW5xdWlzaXRpdmU=")

	testThreeRequestsNotCached(t, req, nil)
}

// Should not cache the response to a request with a `Cookie` header.
func TestNoCacheHeaderCookie(t *testing.T) {
	ResetBackends(backendsByPriority)

	req := NewUniqueEdgeGET(t)
	req.Header.Set("Cookie", "sekret=mekmitasdigoat")

	testThreeRequestsNotCached(t, req, nil)
}

// Should not cache a response with a `Set-Cookie` header.
func TestNoCacheHeaderSetCookie(t *testing.T) {
	ResetBackends(backendsByPriority)

	handler := func(h http.Header) {
		h.Set("Set-Cookie", "sekret=mekmitasdigoat")
	}

	req := NewUniqueEdgeGET(t)
	testThreeRequestsNotCached(t, req, handler)
}

// Should not cache responses with a `Cache-Control: no-cache` header.
// Varnish doesn't respect this by default.
func TestNoCacheCacheControlNoCache(t *testing.T) {
	ResetBackends(backendsByPriority)

	handler := func(h http.Header) {
		h.Set("Cache-Control", "no-cache")
	}

	req := NewUniqueEdgeGET(t)
	testThreeRequestsNotCached(t, req, handler)
}

// Should not cache responses with a `Cache-Control: no-store` header.
// Varnish doesn't respect this by default.
func TestNoCacheCacheControlNoStore(t *testing.T) {
	ResetBackends(backendsByPriority)

	handler := func(h http.Header) {
		h.Set("Cache-Control", "no-store")
	}

	req := NewUniqueEdgeGET(t)
	testThreeRequestsNotCached(t, req, handler)
}

// Should not cache a response with a `Cache-Control: private` header.
func TestNoCacheHeaderCacheControlPrivate(t *testing.T) {
	ResetBackends(backendsByPriority)

	handler := func(h http.Header) {
		h.Set("Cache-Control", "private")
	}

	req := NewUniqueEdgeGET(t)
	testThreeRequestsNotCached(t, req, handler)
}

// Should not cache a response with a `Cache-Control: max-age=0` header.
func TestNoCacheHeaderCacheControlMaxAge0(t *testing.T) {
	ResetBackends(backendsByPriority)

	handler := func(h http.Header) {
		h.Set("Cache-Control", "max-age=0")
	}

	req := NewUniqueEdgeGET(t)
	testThreeRequestsNotCached(t, req, handler)
}

// Should not cache a response with a `Vary: *` header.
func TestNoCacheHeaderVaryAsterisk(t *testing.T) {
	t.Skip("Not widely supported")

	ResetBackends(backendsByPriority)

	handler := func(h http.Header) {
		h.Set("Vary", "*")
	}

	req := NewUniqueEdgeGET(t)
	testThreeRequestsNotCached(t, req, handler)
}
