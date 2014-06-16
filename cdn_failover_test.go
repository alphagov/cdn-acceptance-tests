package main

import (
	"testing"
)

// Should serve a known static error page if all backend servers are down
// and object isn't in cache/stale.
// NB: ideally this should be a page that we control that has a mechanism
//     to alert us that it has been served.
func TestFailoverErrorPageAllServersDown(t *testing.T) {
	t.Error("Not implemented")
}

// Should serve a known static error page if all backend servers return a
// 5xx response and object isn't in cache/stale.
// NB: ideally this should be a page that we control that has a mechanism
//     to alert us that it has been served.
func TestFailoverErrorPageAllServers5xx(t *testing.T) {
	t.Error("Not implemented")
}

// Should serve stale object and not hit mirror(s) if origin is down and
// object is beyond TTL but still in cache.
func TestFailoverOriginDownServeStale(t *testing.T) {
	t.Error("Not implemented")
}

// Should serve stale object and not hit mirror(s) if origin returns a 5xx
// response and object is beyond TTL but still in cache.
func TestFailoverOrigin5xxServeStale(t *testing.T) {
	t.Error("Not implemented")
}

// Should fallback to first mirror if origin is down and object is not in
// cache (active or stale).
func TestFailoverOriginDownUseFirstMirror(t *testing.T) {
	t.Error("Not implemented")
}

// Should fallback to first mirror if origin returns 5xx response and object
// is not in cache (active or stale).
func TestFailoverOrigin5xxUseFirstMirror(t *testing.T) {
	t.Error("Not implemented")
}

// Should fallback to second mirror if both origin and first mirror are
// down.
func TestFailoverOriginDownFirstMirrorDownUseSecondMirror(t *testing.T) {
	t.Error("Not implemented")
}

// Should fallback to second mirror if both origin and first mirror return
// 5xx responses.
func TestFailoverOrigin5xxFirstMirror5xxUseSecondMirror(t *testing.T) {
	t.Error("Not implemented")
}

// Should not fallback to mirror if origin returns a 5xx response with a
// No-Fallback header.
func TestFailoverNoFallbackHeader(t *testing.T) {
	t.Error("Not implemented")
}
