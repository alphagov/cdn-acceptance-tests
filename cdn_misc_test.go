package main

import (
	"fmt"
	"io/ioutil"
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

// Should return 403 and not invalidate the edge's cache for PURGE requests
// that come from IPs not in the whitelist. We assume that this is not
// running from a whitelisted address.
func TestMiscRestrictPurgeRequests(t *testing.T) {
	ResetBackends(backendsByPriority)

	var expectedBody string
	var expectedStatus int
	req := NewUniqueEdgeGET(t)

	for requestCount := 1; requestCount < 4; requestCount++ {
		switch requestCount {
		case 1:
			req.Method = "GET"
			expectedBody = "this should not be purged"
			expectedStatus = 200

			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(expectedBody))
			})
		case 2:
			req.Method = "PURGE"
			expectedBody = ""
			expectedStatus = 403

			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				t.Error("Request should not have made it to origin")
				w.Write([]byte(originServer.Name))
			})
		case 3:
			req.Method = "GET"
			expectedBody = "this should not be purged"
			expectedStatus = 200
		}

		resp := RoundTripCheckError(t, req)

		if resp.StatusCode != expectedStatus {
			t.Errorf(
				"Request %d received incorrect status code. Expected %d, got %d",
				requestCount,
				expectedStatus,
				resp.StatusCode,
			)
		}

		if expectedBody != "" {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if bodyStr := string(body); bodyStr != expectedBody {
				t.Errorf(
					"Request %d received incorrect response body. Expected %q, got %q",
					requestCount,
					expectedBody,
					bodyStr,
				)
			}
		}
	}
}
