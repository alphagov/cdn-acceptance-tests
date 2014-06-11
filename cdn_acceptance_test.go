package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
)

func TestHelpers(t *testing.T) {
	testHelpersCDNServeMuxHandlers(t, originServer)
	testHelpersCDNServeMuxProbes(t, originServer)
}

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

// Should send request to origin by default
func TestRequestsGoToOriginByDefault(t *testing.T) {
	uuid := NewUUID()
	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/%s", uuid) {
			w.Header().Set("EnsureOriginServed", uuid)
		}
	})

	sourceUrl := fmt.Sprintf("https://%s/%s", *edgeHost, uuid)

	req, _ := http.NewRequest("GET", sourceUrl, nil)
	resp, err := client.RoundTrip(req)

	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Status code expected 200, got %d", resp.StatusCode)
	}
	if d := resp.Header.Get("EnsureOriginServed"); d != uuid {
		t.Errorf("EnsureOriginServed header has not come from Origin: expected %q, got %q", uuid, d)
	}

}

// Should cache first response and return it on second request without
// hitting origin again.
func TestFirstResponseCached(t *testing.T) {
	const bodyExpected = "first request"
	const requestsExpectedCount = 1
	requestsReceivedCount := 0

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		if requestsReceivedCount == 0 {
			w.Write([]byte(bodyExpected))
		} else {
			w.Write([]byte("subsequent request"))
		}

		requestsReceivedCount++
	})

	url := fmt.Sprintf("https://%s/%s", *edgeHost, NewUUID())
	req, _ := http.NewRequest("GET", url, nil)

	for i := 0; i < 2; i++ {
		resp, err := client.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != bodyExpected {
			t.Errorf("Incorrect response body. Expected %q, got %q", bodyExpected, body)
		}
	}

	if requestsReceivedCount > requestsExpectedCount {
		t.Errorf("originServer got too many requests. Expected %d requests, got %d", requestsExpectedCount, requestsReceivedCount)
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

// Should serve a known static error page if cannot serve a page
// from origin, stale or any mirror.
// NB: ideally this should be a page that we control that has a mechanism
//     to alert us that it has been served.
func TestErrorPageIsServedWhenNoBackendAvailable(t *testing.T) {
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

// Should set an X-Served-By header giving information on the node and location served from.
func TestXServedByHeaderContainsANodeIdAndLocation(t *testing.T) {
	t.Error("Not implemented")
}

// Should set an X-Cache-Hits header containing hit count for this object,
// from the provider not origin
func TestXCacheHitsContainsProviderHitCountForThisObject(t *testing.T) {
	t.Error("Not implemented")
}
