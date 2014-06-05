package main

import (
	"flag"
	"fmt"
	"testing"
	"net/http"
)

var edgeHost = flag.String("edgeHost", "www.gov.uk", "Hostname of edge")

// Should redirect from HTTP to HTTPS without hitting origin.
func TestProtocolRedirect(t *testing.T) {
	sourceUrl := fmt.Sprintf("http://%s/", *edgeHost)
	destUrl := fmt.Sprintf("https://%s/", *edgeHost)

	client := &http.Transport{}
	req, _ := http.NewRequest("GET", sourceUrl, nil)
	resp, err := client.RoundTrip(req)

	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 301 {
		t.Errorf("Status code expected 301, got %s", resp.StatusCode)
	}
	if d := resp.Header.Get("Location"); d != destUrl {
		t.Errorf("Location header expected %s, got %s", destUrl, d)
	}

	t.Error("Not implemented test to confirm that it doesn't hit origin")
}

// Should return 403 for PURGE requests from IPs not in the whitelist.
func TestRestrictPurgeRequests(t *testing.T) {
	t.Error("Not implemented")
}

// Should create an X-Forwarded-For header containing the client's IP.
func TestHeaderCreateXFF(t *testing.T) {
	t.Error("Not implemented")
}

// Should append client's IP to existing X-Forwarded-For header.
func TestHeaderAppendXFF(t *testing.T) {
	t.Error("Not implemented")
}

// Should create a True-Client-IP header containing the client's IP
// address, discarding the value provided in the original request.
func TestHeaderUnspoofableClientIP(t *testing.T) {
	t.Error("Not implemented")
}

// Should not modify Host header from original request.
func TestHeaderHostUnmodified(t *testing.T) {
	t.Error("Not implemented")
}

// Should set a default TTL if the response doesn't set one.
func TestDefaultTTL(t *testing.T) {
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

// Should not cache a response with a Set-Cookie a header.
func TestNoCacheHeaderSetCookie(t *testing.T) {
	t.Error("Not implemented")
}

// Should not cache a response with a Cache-Control: private header.
func TestNoCacheHeaderCacheControlPrivate(t *testing.T) {
	t.Error("Not implemented")
}
