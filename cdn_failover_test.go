package main

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"
)

// Should serve a known static error page if all backend servers are down
// and object isn't in cache/stale.
// NB: ideally this should be a page that we control that has a mechanism
//     to alert us that it has been served.
func TestFailoverErrorPageAllServersDown(t *testing.T) {
	originServer.Stop()
	backupServer1.Stop()
	backupServer2.Stop()
	time.Sleep(5 * time.Second)

	sourceUrl := fmt.Sprintf("https://%s/?cache-bust=%s", *edgeHost, NewUUID())
	req, err := http.NewRequest("GET", sourceUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 503 {
		t.Errorf("Invalid StatusCode received. Expected 503, got %d", resp.StatusCode)
	}

	err = StartBackendsInOrder(*edgeHost)
	if err != nil {
		// Bomb out - we do not have a consistent backend, so subsequent tests
		// would fail in unexpected ways.
		log.Fatal(err)
	}

}

// Should serve a known static error page if all backend servers return a
// 5xx response and object isn't in cache/stale.
// NB: ideally this should be a page that we control that has a mechanism
//     to alert us that it has been served.
func TestFailoverErrorPageAllServers5xx(t *testing.T) {
	t.Error("Not implemented")
}

// Should back off requests against origin for a very short period of time
// if origin returns a 5xx response so as not to overwhelm it.
func TestFailoverOrigin5xxBackOff(t *testing.T) {
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
