package main

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"testing"
)

// Should redirect from HTTP to HTTPS without hitting origin.
func TestProtocolRedirect(t *testing.T) {
	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have made it to origin")
	})

	sourceUrl := fmt.Sprintf("http://%s/foo/bar", *edgeHost)
	destUrl := fmt.Sprintf("https://%s/foo/bar", *edgeHost)

	req, _ := http.NewRequest("GET", sourceUrl, nil)
	resp, err := client.RoundTrip(req)

	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 301 {
		t.Errorf("Status code expected 301, got %d", resp.StatusCode)
	}
	if d := resp.Header.Get("Location"); d != destUrl {
		t.Errorf("Location header expected %s, got %s", destUrl, d)
	}
}

// Should return 403 for PURGE requests from IPs not in the whitelist. We
// assume that this is not running from a whitelisted address.
func TestRestrictPurgeRequests(t *testing.T) {
	const expectedStatusCode = 403

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have made it to origin")
	})

	url := fmt.Sprintf("https://%s/", *edgeHost)
	req, _ := http.NewRequest("PURGE", url, nil)

	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != expectedStatusCode {
		t.Errorf("Incorrect status code. Expected %d, got %d", expectedStatusCode, resp.StatusCode)
	}
}

// Should set an `X-Forwarded-For` header for requests that don't already
// have one and append to requests that already have the header. This test
// will not work if run from behind a proxy that also sets XFF.
func TestHeaderXFFCreateAndAppend(t *testing.T) {
	const headerName = "X-Forwarded-For"
	const sentHeaderVal = "203.0.113.99"
	var ourReportedIP net.IP
	var receivedHeaderVal string

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaderVal = r.Header.Get(headerName)
	})

	url := fmt.Sprintf("https://%s/%s", *edgeHost, NewUUID())
	req, _ := http.NewRequest("GET", url, nil)

	// First request with no existing XFF.
	_, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if receivedHeaderVal == "" {
		t.Fatalf("Origin didn't receive request with %q header", headerName)
	}

	ourReportedIP = net.ParseIP(receivedHeaderVal)
	if ourReportedIP == nil {
		t.Fatalf(
			"Expected origin to receive %q header with single IP. Got %q",
			headerName,
			receivedHeaderVal,
		)
	}

	// Use the IP returned by the first response to predict the second.
	expectedHeaderVal := fmt.Sprintf("%s, %s", sentHeaderVal, ourReportedIP.String())

	// Second request with existing XFF.
	url = fmt.Sprintf("https://%s/%s", *edgeHost, NewUUID())
	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Set(headerName, sentHeaderVal)

	_, err = client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}

	if receivedHeaderVal != expectedHeaderVal {
		t.Errorf(
			"Origin received %q header with wrong value. Expected %q, got %q",
			headerName,
			expectedHeaderVal,
			receivedHeaderVal,
		)
	}
}

// Should create a True-Client-IP header containing the client's IP
// address, discarding the value provided in the original request.
func TestHeaderUnspoofableClientIP(t *testing.T) {
	const headerName = "True-Client-IP"
	const sentHeaderVal = "203.0.113.99"
	var sentHeaderIP = net.ParseIP(sentHeaderVal)
	var receivedHeaderVal string

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaderVal = r.Header.Get(headerName)
	})

	url := fmt.Sprintf("https://%s/%s", *edgeHost, NewUUID())
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set(headerName, sentHeaderVal)

	_, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}

	receivedHeaderIP := net.ParseIP(receivedHeaderVal)
	if receivedHeaderIP == nil {
		t.Fatalf("Origin received %q header with non-IP value %q", headerName, receivedHeaderVal)
	}
	if receivedHeaderIP.Equal(sentHeaderIP) {
		t.Errorf("Origin received %q header with unmodified value %q", headerName, receivedHeaderIP)
	}
}

// Should not modify `Host` header from original request.
func TestHeaderHostUnmodified(t *testing.T) {
	const headerName = "Host"
	var sentHeaderVal = *edgeHost
	var receivedHeaderVal string

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaderVal = r.Host
	})

	url := fmt.Sprintf("https://%s/%s", sentHeaderVal, NewUUID())
	req, _ := http.NewRequest("GET", url, nil)

	if req.Host != sentHeaderVal {
		t.Errorf(
			"Constructed request contains wrong %q header. Expected %q, got %q",
			headerName,
			sentHeaderVal,
			req.Host,
		)
	}

	_, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}

	if receivedHeaderVal != sentHeaderVal {
		t.Errorf(
			"Origin received %q header with modified value. Expected %q, got %q",
			headerName,
			sentHeaderVal,
			receivedHeaderVal,
		)
	}
}

// Should serve a known static error page if cannot serve a page
// from origin, stale or any mirror.
// NB: ideally this should be a page that we control that has a mechanism
//     to alert us that it has been served.
func TestErrorPageIsServedWhenNoBackendAvailable(t *testing.T) {
	t.Error("Not implemented")
}

// ---------------------------------------------------------
// Test that useful common cache-related parameters are sent to the
// client by this CDN provider.

// Should set an Age header itself rather than passing the Age header from origin.
func TestAgeHeaderIsSetByProviderNotOrigin(t *testing.T) {
	t.Error("Not implemented")
}

// Should set an X-Cache header containing HIT/MISS from 'origin, itself'
func TestXCacheHeaderContainsHitMissFromBothProviderAndOrigin(t *testing.T) {
	t.Error("Not implemented")
}

// Should set an X-Served-By header giving information on the (Fastly) node and location served from.
func TestXServedByHeaderContainsFastlyNodeIdAndLocation(t *testing.T) {

	expectedFastlyXServedByRegexp := regexp.MustCompile("^cache-[a-z0-9]+-[A-Z]{3}$")

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {})

	sourceUrl := fmt.Sprintf("https://%s/", *edgeHost)

	req, _ := http.NewRequest("GET", sourceUrl, nil)
	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}

	actualHeader := resp.Header.Get("X-Served-By")
	if actualHeader == "" {
		t.Error("X-Served-By header has not been set by Edge")
	}

	if expectedFastlyXServedByRegexp.FindString(actualHeader) != actualHeader {
		t.Errorf("X-Served-By is not as expected: got %q", actualHeader)
	}

}

// Should set an X-Cache-Hits header containing hit count for this object,
// from the Edge AND the Origin, assuming Origin sets one.
// This is in the format "{origin-hit-count}, {edge-hit-count}"
func TestXCacheHitsContainsProviderHitCountForThisObject(t *testing.T) {

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
	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}

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
	resp, err = client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}

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
