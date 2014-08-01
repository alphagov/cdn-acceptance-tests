package main

import (
	"fmt"
	"net"
	"net/http"
	"testing"
)

// Should set an `X-Forwarded-For` header for requests that don't already
// have one and append to requests that already have the header. This test
// will not work if run from behind a proxy that also sets XFF.
func TestReqHeaderXFFCreateAndAppend(t *testing.T) {
	ResetBackends(backendsByPriority)

	const headerName = "X-Forwarded-For"
	const sentHeaderVal = "203.0.113.99"
	var ourReportedIP net.IP
	var receivedHeaderVal string

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaderVal = r.Header.Get(headerName)
	})

	// First request with no existing XFF.
	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

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
	req = NewUniqueEdgeGET(t)
	req.Header.Set(headerName, sentHeaderVal)

	resp = RoundTripCheckError(t, req)
	defer resp.Body.Close()

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
func TestReqHeaderUnspoofableClientIP(t *testing.T) {
	ResetBackends(backendsByPriority)

	const headerName = "True-Client-IP"
	const sentHeaderVal = "203.0.113.99"
	var sentHeaderIP = net.ParseIP(sentHeaderVal)
	var receivedHeaderVal string

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaderVal = r.Header.Get(headerName)
	})

	req := NewUniqueEdgeGET(t)
	req.Header.Set(headerName, sentHeaderVal)

	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	receivedHeaderIP := net.ParseIP(receivedHeaderVal)
	if receivedHeaderIP == nil {
		t.Fatalf("Origin received %q header with non-IP value %q", headerName, receivedHeaderVal)
	}
	if receivedHeaderIP.Equal(sentHeaderIP) {
		t.Errorf("Origin received %q header with unmodified value %q", headerName, receivedHeaderIP)
	}
}

// Should not modify `Host` header from original request.
func TestReqHeaderHostUnmodified(t *testing.T) {
	const headerName = "Host"
	var sentHeaderVal = *edgeHost
	var receivedHeaderVal string

	ResetBackends(backendsByPriority)
	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaderVal = r.Host
	})

	req := NewUniqueEdgeGET(t)

	if req.Host != sentHeaderVal {
		t.Errorf(
			"Constructed request contains wrong %q header. Expected %q, got %q",
			headerName,
			sentHeaderVal,
			req.Host,
		)
	}

	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if receivedHeaderVal != sentHeaderVal {
		t.Errorf(
			"Origin received %q header with modified value. Expected %q, got %q",
			headerName,
			sentHeaderVal,
			receivedHeaderVal,
		)
	}
}
