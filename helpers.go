package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// CDNBackendServer is a backend server which will receive and respond to
// requests from the CDN.
type CDNBackendServer struct {
	Name     string
	Port     int
	TLSCerts []tls.Certificate
	handler  func(w http.ResponseWriter, r *http.Request)
	server   *httptest.Server
}

// ServeHTTP satisfies the http.HandlerFunc interface. Health check requests
// for `HEAD` are always served 200 responses. Other requests are passed
// off to a custom handler provided by SwitchHandler.
func (s *CDNBackendServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Backend-Name", s.Name)

        // swallow healtheck requests
	if r.Method == "HEAD" {
		w.Header().Set("PING", "PONG")
		return
	}

	s.handler(w, r)
}

// ResetHandler sets the handler back to an empty function that will return
// a 200 response.
func (s *CDNBackendServer) ResetHandler() {
	s.handler = func(w http.ResponseWriter, r *http.Request) {}
}

// SwitchHandler sets the handler to a custom function. This is used by
// tests to pass in their own request inspection and response handler.
func (s *CDNBackendServer) SwitchHandler(h func(w http.ResponseWriter, r *http.Request)) {
	s.handler = h
}

// IsStarted checks whether the server is currently started.
func (s *CDNBackendServer) IsStarted() bool {
	return (s.server != nil)
}

// Stop closes all outstanding client connections and unbind the port.
// Resets server back to nil, as if the backend had been instantiated but
// Start() not called.
func (s *CDNBackendServer) Stop() {
	s.server.Close()
	s.server = nil
}

// Start resets the handler back to the default and starts the server on
// Port. It will exit immediately if it's unable to bind the port, due to
// permissions or a conflicting application.
func (s *CDNBackendServer) Start() {
	s.ResetHandler()

	addr := fmt.Sprintf(":%d", s.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	// Store the port randomly assigned by the kernel if we started with 0.
	if s.Port == 0 {
		_, portStr, _ := net.SplitHostPort(ln.Addr().String())
		s.Port, _ = strconv.Atoi(portStr)
	}

	s.server = httptest.NewUnstartedServer(s)
	s.server.Listener = ln

	if len(s.TLSCerts) > 0 {
		s.server.TLS = &tls.Config{
			Certificates: s.TLSCerts,
		}
	}

	s.server.StartTLS()
	log.Printf("Started server on port %d", s.Port)
}

// CachedHostLookup caches DNS lookups for the given `Host` in order to
// prevent us switching to another edge location in the middle of tests.
type CachedHostLookup struct {
	Host         string
	hardCachedIP string
}

// lookup performs a DNS lookup and caches the first IP address returned.
// Subsequent requests always return the cached address, preventing further
// DNS requests.
func (c *CachedHostLookup) lookup(host string) string {
	if c.hardCachedIP == "" {
		ipAddresses, err := net.LookupHost(host)
		if err != nil {
			log.Fatal(err)
		}

		c.hardCachedIP = ipAddresses[0]
	}

	return c.hardCachedIP
}

// Dial acts as a wrapper for `net.Dial`, ostensibly for use with
// `http.Transport`. If the hostname matches `Host` then it will use the
// cached address.
func (c *CachedHostLookup) Dial(network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Fatal(err)
	}

	if host != c.Host {
		return net.Dial(network, addr)
	}

	ipAddr := c.lookup(host)
	return net.Dial(network, net.JoinHostPort(ipAddr, port))
}

// NewCachedDial returns the `Dial` function for a new CachedHostLookup
// object with the given host.
func NewCachedDial(host string) func(string, string) (net.Conn, error) {
	c := CachedHostLookup{
		Host: host,
	}

	return c.Dial
}

// NewUUID returns a v4 (random) UUID string.
// This might not be strictly RFC4122 compliant, but it will do. Credit:
// https://groups.google.com/d/msg/golang-nuts/Rn13T6BZpgE/dBaYVJ4hB5gJ
func NewUUID() string {
	bs := make([]byte, 16)
	rand.Read(bs)
	bs[6] = (bs[6] & 0x0f) | 0x40
	bs[8] = (bs[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", bs[0:4], bs[4:6], bs[6:8], bs[8:10], bs[10:])
}

// NewUniqueEdgeURL constructs a new URL for edge. Always uses HTTPS. A random
// UUID is used in the path to ensure that it hasn't previously been cached. It
// is passed as a query param for / so that some of the tests can be run
// against a service that hasn't been configured to point at our test backends.
func NewUniqueEdgeURL() string {
	url := url.URL{
		Scheme: "https",
		Host:   *edgeHost,
		Path:   "/",
		RawQuery: url.Values{
			"nocache": []string{NewUUID()},
		}.Encode(),
	}

	return url.String()
}

// NewUniqueEdgeGET constructs a GET request (but not perform it) against edge.
// Uses NewUniqueEdgeURL() to ensure that it hasn't previously been cached. The
// request method field of the returned object can be later modified if
// required.
func NewUniqueEdgeGET(t *testing.T) *http.Request {
	url := NewUniqueEdgeURL()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}

	return req
}

// RoundTripCheckError makes an HTTP request using http.RoundTrip, which
// doesn't handle redirects or cookies, and return the response. If there are
// any errors then the calling test will be aborted so as not to operate on a
// nil response.
func RoundTripCheckError(t *testing.T, req *http.Request) *http.Response {
	start := time.Now()
	resp, err := client.RoundTrip(req)
	if duration := time.Since(start); duration > requestSlowThreshold {
		t.Error("Slow request, took:", duration)
	}
	if *debugResp {
		t.Logf("%#v", resp)
	}
	if err != nil {
		t.Fatal(err)
	}

	return resp
}

// ResetBackends resets all backends, ensuring that they are started, have the
// default handler function, and that the edge considers them healthy. It may
// take some time because we need to receive and respond to enough probe health
// checks to be considered up.
func ResetBackends(backends []*CDNBackendServer) {
	remainingBackendsStopped := false

	// Reverse priority order so that waitForBackend works.
	for i := len(backends); i > 0; i-- {
		backend := backends[i-1]

		if backend.IsStarted() {
			backend.ResetHandler()
		} else {
			if !remainingBackendsStopped {
				// Ensure all remaining unchecked backends are stopped so that
				// waitForBackend will work. We'll bring them back one-by-one.
				stopBackends(backends[0 : i-1])
				remainingBackendsStopped = true
			}

			backend.Start()
			err := waitForBackend(backend.Name)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

// Ensure that a slice of backends are stopped.
func stopBackends(backends []*CDNBackendServer) {
	for _, backend := range backends {
		if backend.IsStarted() {
			backend.Stop()
		}
	}
}

// Wait for the backend to return with the header we expect. This is designed to
// confirm that requests are hitting this specific backend, rather than a lower-level
// backend that this overrides (for example, origin over a mirror)
//
func waitForBackend(expectedBackendName string) error {
	const maxRetries = 20
	const waitForCdnProbeToPropagate = time.Duration(5 * time.Second)
	const timeBetweenAttempts = time.Duration(2 * time.Second)

	var url string

	log.Printf("Checking health of %s...", expectedBackendName)
	for try := 0; try <= maxRetries; try++ {
		url = NewUniqueEdgeURL()
		req, _ := http.NewRequest("GET", url, nil)

		resp, err := client.RoundTrip(req)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.Header.Get("Backend-Name") == expectedBackendName {
			if try != 0 {
				time.Sleep(waitForCdnProbeToPropagate)
			}

			log.Println(expectedBackendName + " is up!")
			return nil // all is well!
		}

		time.Sleep(timeBetweenAttempts)
	}

	return fmt.Errorf(
		"%s still not available after %d attempts",
		expectedBackendName,
		maxRetries,
	)

}

// Callback function to modify complete response.
type responseCallback func(w http.ResponseWriter)

// Wrapper for testRequestsCachedDuration() with a respTTL of zero.
// Meaning that the cached object doesn't expire.
func testRequestsCachedIndefinite(
	t *testing.T,
	req *http.Request,
	respCB responseCallback,
) {
	testRequestsCachedDuration(t, req, respCB, time.Duration(0))
}

// Helper function to make three requests and test responses. If respTTL is:
//
//	- zero: no delay between requests, origin should only see one request,
//		and all response bodies should be identical (from cache).
//	- non-zero: first and second request without delay, origin should only
//		see one request and responses bodies should be identical, then after a
//		delay of respTTL + a buffer a third response should get a new response
//		directly from origin.
//
// A responseCallback, if not nil, will be called to modify the response
// before calling Write(body).
func testRequestsCachedDuration(
	t *testing.T,
	req *http.Request,
	respCB responseCallback,
	respTTL time.Duration,
) {
	const responseCached = "first response"
	const responseNotCached = "subsequent response"
	var testCacheExpiry = respTTL > 0
	var respTTLWithBuffer = respTTL + (respTTL / 4)
	var requestsExpectedCount int

	requestsReceivedCount := 0
	switch testCacheExpiry {
	case true:
		requestsExpectedCount = 2
	case false:
		requestsExpectedCount = 1
	}

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		if respCB != nil {
			respCB(w)
		}

		if requestsReceivedCount == 0 {
			w.Write([]byte(responseCached))
		} else {
			w.Write([]byte(responseNotCached))
		}

		requestsReceivedCount++
	})

	for requestCount := 1; requestCount < 4; requestCount++ {
		if testCacheExpiry && requestCount == 3 {
			time.Sleep(respTTLWithBuffer)
		}

		resp := RoundTripCheckError(t, req)
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var expectedBody string
		if testCacheExpiry && requestCount > 2 {
			expectedBody = responseNotCached
		} else {
			expectedBody = responseCached
		}

		if receivedBody := string(body); receivedBody != expectedBody {
			t.Errorf(
				"Request %d received incorrect response body. Expected %q, got %q",
				requestCount,
				expectedBody,
				receivedBody,
			)
		}
	}

	if requestsReceivedCount != requestsExpectedCount {
		t.Errorf(
			"Origin received the wrong number of requests. Expected %d, got %d",
			requestsExpectedCount,
			requestsReceivedCount,
		)
	}
}

// Callback function to modify response headers.
type responseHeaderCallback func(h http.Header)

// Helper function to make three requests and verify that we get three
// unique and uncached responses back. A responseHeaderCallback, if not nil,
// will be called to modify the response headers.
func testThreeRequestsNotCached(t *testing.T, req *http.Request, headerCB responseHeaderCallback) {
	requestsReceivedCount := 0
	responseBodies := []string{
		"first response",
		"second response",
		"third response",
	}

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		if headerCB != nil {
			headerCB(w.Header())
		}
		w.Write([]byte(responseBodies[requestsReceivedCount]))
		requestsReceivedCount++
	})

	for requestCount, expectedBody := range responseBodies {
		resp := RoundTripCheckError(t, req)
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if receivedBody := string(body); receivedBody != expectedBody {
			t.Errorf(
				"Request %d received incorrect response body. Expected %q, got %q",
				requestCount+1,
				expectedBody,
				receivedBody,
			)
		}
	}
}

// testResponseNotManipulated configures origin to respond to a request with
// the contents of fixture file. It then makes a request and asserts that
// the response body matches the original fixture file, meaning that the CDN
// hasn't manipulated it in any way. The `Content-Type` and request path are
// set according to the fixture's file extension to ensure that the CDN
// detects it correctly.
func testResponseNotManipulated(t *testing.T, fixtureFile string) {
	fixtureData, err := ioutil.ReadFile(fixtureFile)
	if err != nil {
		t.Fatalf("Unable load fixture file %q", fixtureFile)
	}

	contentType := mime.TypeByExtension(filepath.Ext(fixtureFile))
	if contentType == "" || strings.Contains(contentType, "text/plain") {
		t.Fatalf("Unable to determine fixture Content-Type. Got %q", contentType)
	}

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Write(fixtureData)
	})

	req := NewUniqueEdgeGET(t)
	req.URL.Path = "/" + filepath.Base(fixtureFile)

	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(body, fixtureData) {
		t.Error("Response body did not match fixture")

		if bytes.Compare(body, fixtureData) != 0 {
			t.Errorf(
				"Response body sizes for debug purposes. Expected %d, got %d",
				len(fixtureData),
				len(body),
			)
		}
	}
}
