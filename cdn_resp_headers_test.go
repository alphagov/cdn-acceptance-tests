package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"
)

// Test that useful common cache-related parameters are sent to the
// client by this CDN provider.

// Should propagate an Age header from origin and then increment it for the
// time it's in cache.
func TestRespHeaderAge(t *testing.T) {
	ResetBackends(backendsByPriority)

	const originAgeInSeconds = 100
	const secondsToWaitBetweenRequests = 5
	const expectedAgeInSeconds = originAgeInSeconds + secondsToWaitBetweenRequests
	requestReceivedCount := 0

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		if requestReceivedCount == 0 {
			w.Header().Set("Cache-Control", "max-age=1800, public")
			w.Header().Set("Age", fmt.Sprintf("%d", originAgeInSeconds))
			w.Write([]byte("cacheable request"))
		} else {
			t.Error("Unexpected subsequent request received at Origin")
		}
		requestReceivedCount++
	})

	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Edge returned an unexpected status: %q", resp.Status)
	}

	// wait a little bit. Edge should update the Age header, we know Origin will not
	time.Sleep(time.Duration(secondsToWaitBetweenRequests) * time.Second)
	resp = RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Edge returned an unexpected status: %q", resp.Status)
	}

	edgeAgeHeader := resp.Header.Get("Age")
	if edgeAgeHeader == "" {
		t.Fatal("Age Header is not set")
	}

	edgeAgeInSeconds, convErr := strconv.Atoi(edgeAgeHeader)
	if convErr != nil {
		t.Fatal(convErr)
	}

	if edgeAgeInSeconds != expectedAgeInSeconds {
		t.Errorf(
			"Age header from Edge is not as expected. Got %q, expected '%d'",
			edgeAgeHeader,
			expectedAgeInSeconds,
		)
	}
}

// Should set an X-Served-By header giving information on the (Fastly) node and location served from.
func TestRespHeaderXServedBy(t *testing.T) {
	ResetBackends(backendsByPriority)

	expectedFastlyXServedByRegexp := regexp.MustCompile("^cache-[a-z0-9]+-[A-Z]{3}$")

	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	actualHeader := resp.Header.Get("X-Served-By")
	if actualHeader == "" {
		t.Error("X-Served-By header has not been set by Edge")
	}

	if expectedFastlyXServedByRegexp.FindString(actualHeader) != actualHeader {
		t.Errorf("X-Served-By is not as expected: got %q", actualHeader)
	}

}
