package main

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"
)

// Test that useful common cache-related parameters are sent to the
// client by this CDN provider.

// Should propagate an Age header from origin and then increment it for the
// time it is in edge's cache. This assumes no request/response delay:
// http://tools.ietf.org/html/rfc7234#section-4.2.3
func TestRespHeaderAgeFromOrigin(t *testing.T) {
	ResetBackends(backendsByPriority)

	const originAgeInSeconds = 100
	const secondsToWaitBetweenRequests = 5
	expectedHeaderVals := []string{
		fmt.Sprintf("%d", originAgeInSeconds),
		fmt.Sprintf("%d", originAgeInSeconds + secondsToWaitBetweenRequests),
	}

	req := NewUniqueEdgeGET(t)

	for requestCount, expectedHeaderVal := range expectedHeaderVals {
		requestCount = requestCount + 1
		switch requestCount {
		case 1:
			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Age", expectedHeaderVal)
				w.Header().Set("Cache-Control", "max-age=1800, public")
				w.Write([]byte("cacheable request"))
			})
		case 2:
			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				t.Error("Origin received request and it shouldn't have")
			})

			// Wait for Age to increment.
			time.Sleep(time.Duration(secondsToWaitBetweenRequests) * time.Second)
		}

		resp := RoundTripCheckError(t, req)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Request %d received incorrect status %q", requestCount, resp.Status)
		}

		if val := resp.Header.Get("Age"); val != expectedHeaderVal {
			t.Errorf(
				"Request %d received incorrect Age header. Got %q, expected %q",
				requestCount,
				val,
				expectedHeaderVal,
			)
		}
	}
}

// Should set an X-Cache header containing HIT/MISS from 'origin, itself'
func TestRespHeaderXCacheAppend(t *testing.T) {
	ResetBackends(backendsByPriority)

	if vendorCloudflare {
		t.Skip(notSupportedByVendor)
	}

	const originXCache = "HIT"

	var (
		xCache         string
		expectedXCache string
	)

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Cache", originXCache)
	})

	// Get first request, will come from origin, cannot be cached - hence cache MISS
	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	xCache = resp.Header.Get("X-Cache")
	expectedXCache = fmt.Sprintf("%s, MISS", originXCache)
	if xCache != expectedXCache {
		t.Errorf(
			"X-Cache on initial hit is wrong: expected %q, got %q",
			expectedXCache,
			xCache,
		)
	}

}

// Should set a header containing 'HIT' or 'MISS' depending on whether request is cached
func TestRespHeaderCacheHitMiss(t *testing.T) {
	ResetBackends(backendsByPriority)

	var (
		headerName  string
		headerValue string
	)

	switch {
	case vendorCloudflare:
		headerName = "CF-Cache-Status"
	case vendorFastly:
		headerName = "X-Cache"
	default:
		t.Fatal(notImplementedForVendor)
	}

	expectedHeaderValues := []string{"MISS", "HIT"}
	const cacheDuration = time.Second

	if vendorCloudflare {
		cloudFlareStatuses := []string{"EXPIRED", "HIT"}
		expectedHeaderValues = append(expectedHeaderValues, cloudFlareStatuses...)
	}

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		cacheControlValue := fmt.Sprintf("max-age=%.0f", cacheDuration.Seconds())
		w.Header().Set("Cache-Control", cacheControlValue)
	})

	req := NewUniqueEdgeGET(t)

	for count, expectedValue := range expectedHeaderValues {

		if expectedValue == "EXPIRED" {
			// sleep long enough for object to have expired
			sleepDuration := cacheDuration + time.Second
			time.Sleep(sleepDuration)
		}

		resp := RoundTripCheckError(t, req)
		defer resp.Body.Close()

		headerValue = resp.Header.Get(headerName)

		if headerValue != expectedValue {
			t.Errorf(
				"%s on request %d is wrong: expected %q, got %q",
				headerName,
				count+1,
				expectedValue,
				headerValue,
			)
		}
	}
}

// Should set an 'Served-By' header giving information on the edge node and location served from.
func TestRespHeaderServedBy(t *testing.T) {
	ResetBackends(backendsByPriority)

	var expectedServedByRegexp *regexp.Regexp
	var headerName string

	switch {
	case vendorCloudflare:
		headerName = "CF-RAY"
		expectedServedByRegexp = regexp.MustCompile("^[a-z0-9]{16}-[A-Z]{3}$")
	case vendorFastly:
		headerName = "X-Served-By"
		expectedServedByRegexp = regexp.MustCompile("^cache-[a-z0-9]+-[A-Z]{3}$")
	default:
		t.Fatal(notImplementedForVendor)
	}

	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	actualHeader := resp.Header.Get(headerName)

	if actualHeader == "" {
		t.Error(headerName + " header has not been set by Edge")
	}

	if expectedServedByRegexp.FindString(actualHeader) != actualHeader {
		t.Errorf("%s is not as expected: got %q", headerName, actualHeader)
	}

}

// Should set an X-Cache-Hits header containing hit count for this object,
// from the Edge AND the Origin, assuming Origin sets one.
// This is in the format "{origin-hit-count}, {edge-hit-count}"
func TestRespHeaderXCacheHitsAppend(t *testing.T) {
	ResetBackends(backendsByPriority)

	if vendorCloudflare {
		t.Skip(notSupportedByVendor)
	}

	const originXCacheHits = "53"

	var (
		xCacheHits         string
		expectedXCacheHits string
	)

	uuid := NewUUID()

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/%s", uuid) {
			w.Header().Set("X-Cache-Hits", originXCacheHits)
		}
	})

	sourceUrl := fmt.Sprintf("https://%s/%s", *edgeHost, uuid)

	// Get first request, will come from origin. Edge Hit Count 0
	req, _ := http.NewRequest("GET", sourceUrl, nil)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	xCacheHits = resp.Header.Get("X-Cache-Hits")
	expectedXCacheHits = fmt.Sprintf("%s, 0", originXCacheHits)
	if xCacheHits != expectedXCacheHits {
		t.Errorf(
			"X-Cache-Hits on initial hit is wrong: expected %q, got %q",
			expectedXCacheHits,
			xCacheHits,
		)
	}

	// Get request again. Should come from Edge now, hit count 1
	resp = RoundTripCheckError(t, req)
	defer resp.Body.Close()

	xCacheHits = resp.Header.Get("X-Cache-Hits")
	expectedXCacheHits = fmt.Sprintf("%s, 1", originXCacheHits)
	if xCacheHits != expectedXCacheHits {
		t.Errorf(
			"X-Cache-Hits on second hit is wrong: expected %q, got %q",
			expectedXCacheHits,
			xCacheHits,
		)
	}
}
