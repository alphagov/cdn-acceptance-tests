package main

import (
	"fmt"
	"net/http"
	"testing"
)

// Should redirect from HTTP to HTTPS without hitting origin.
func TestMiscProtocolRedirect(t *testing.T) {
	ResetBackends(backendsByPriority)

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have made it to origin")
	})

	sourceUrl := fmt.Sprintf("http://%s/foo/bar", *edgeHost)
	destUrl := fmt.Sprintf("https://%s/foo/bar", *edgeHost)

	req, _ := http.NewRequest("GET", sourceUrl, nil)
	resp := RoundTripCheckError(t, req)

	if resp.StatusCode != 301 {
		t.Errorf("Status code expected 301, got %d", resp.StatusCode)
	}
	if d := resp.Header.Get("Location"); d != destUrl {
		t.Errorf("Location header expected %s, got %s", destUrl, d)
	}
}

// Should return 403 for PURGE requests from IPs not in the whitelist. We
// assume that this is not running from a whitelisted address.
func TestMiscRestrictPurgeRequests(t *testing.T) {
	ResetBackends(backendsByPriority)

	const expectedStatusCode = 403

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have made it to origin")
	})

	req := NewUniqueEdgeGET(t)
	req.Method = "PURGE"
	resp := RoundTripCheckError(t, req)

	if resp.StatusCode != expectedStatusCode {
		t.Errorf("Incorrect status code. Expected %d, got %d", expectedStatusCode, resp.StatusCode)
	}
}
