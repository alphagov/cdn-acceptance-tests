package main

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// An instance of a backend server which will receive requests from the CDN.
// Implements the http.Handler interface with a modifiable handler so that
// tests can pass in their own functions to inspect requests and modify
// responses.
type CDNBackendServer struct {
	Name    string
	Port    int
	handler func(w http.ResponseWriter, r *http.Request)
	server  *httptest.Server
}

func (s *CDNBackendServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Backend-Name", s.Name)
	if r.Method == "HEAD" && r.URL.Path == "/" {
		w.Header().Set("PING", "PONG")
		return
	}

	s.handler(w, r)
}

func (s *CDNBackendServer) ResetHandler() {
	s.handler = func(w http.ResponseWriter, r *http.Request) {}
}

func (s *CDNBackendServer) SwitchHandler(h func(w http.ResponseWriter, r *http.Request)) {
	s.handler = h
}

func (s *CDNBackendServer) Stop() {
	s.server.Close()
}

func (s *CDNBackendServer) Start() {
	s.ResetHandler()
	addr := fmt.Sprintf(":%d", s.Port)

	go func() {
		err := StoppableHttpListenAndServe(addr, s)
		if err != nil {
			log.Fatal(err)
		}
	}()

	log.Printf("Started server on port %d", s.Port)
}

func StoppableHttpListenAndServe(addr string, backend *CDNBackendServer) error {
	server := httptest.NewUnstartedServer(backend)
	backend.server = server

	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatal(e)
	}

	server.Listener = l
	server.Start()
	return nil
}

// Return a v4 (random) UUID string.
// This might not be strictly RFC4122 compliant, but it will do. Credit:
// https://groups.google.com/d/msg/golang-nuts/Rn13T6BZpgE/dBaYVJ4hB5gJ
func NewUUID() string {
	bs := make([]byte, 16)
	rand.Read(bs)
	bs[6] = (bs[6] & 0x0f) | 0x40
	bs[8] = (bs[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", bs[0:4], bs[4:6], bs[6:8], bs[8:10], bs[10:])
}

// Construct a new URL for edge. Always uses HTTPS. A random UUID is used in
// the path to ensure that it hasn't previously been cached. It is passed as
// a query param for / so that some of the tests can be run against a
// service that hasn't been configured to point at our test backends.
func NewUniqueEdgeURL() string {
	url := url.URL{
		Scheme: "https",
		Host:   *edgeHost,
		Path:   fmt.Sprintf("/?nocache=%s", NewUUID()),
	}

	return url.String()
}

// Construct a GET request (but not perform it) against edge. Uses
// NewUniqueEdgeURL() to ensure that it hasn't previously been cached. The
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

// Make an HTTP request using http.RoundTrip, which doesn't handle redirects
// or cookies, and return the response. If there are any errors then the
// calling test will be aborted so as not to operate on a nil response.
func RoundTripCheckError(t *testing.T, req *http.Request) *http.Response {
	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}

	return resp
}

// Confirm that the edge (CDN) is working correctly with respect to its
// perception of the state of its backend nodes. This may take some time
// because our CDNBackendServer needs to receive and respond to enough probe
// health checks to be considered up.
//
// We assume that all backends are stopped, so that we can start them in order.
//
func StartBackendsInOrder(edgeHost string, backends []*CDNBackendServer) {
	for i := len(backends); i > 0; i-- {
		backends[i-1].Start()
		err := waitForBackend(edgeHost, backends[i-1].Name)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// Wait for the backend to return with the header we expect. This is designed to
// confirm that requests are hitting this specific backend, rather than a lower-level
// backend that this overrides (for example, origin over a mirror)
//
func waitForBackend(
	edgeHost string,
	expectedBackendName string,
) error {

	const maxRetries = 20
	const waitForCdnProbeToPropagate = time.Duration(5 * time.Second)
	const timeBetweenAttempts = time.Duration(2 * time.Second)

	var sourceUrl string

	log.Printf("Checking health of %s...", expectedBackendName)
	for try := 0; try <= maxRetries; try++ {
		uuid := NewUUID()
		sourceUrl = fmt.Sprintf("https://%s/?cacheBuster=%s", edgeHost, uuid)
		req, _ := http.NewRequest("GET", sourceUrl, nil)
		resp, err := client.RoundTrip(req)
		if err != nil {
			return err
		}
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
func testRequestsCachedIndefinite(t *testing.T, respCB responseCallback) {
	testRequestsCachedDuration(t, respCB, time.Duration(0))
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
func testRequestsCachedDuration(t *testing.T, respCB responseCallback, respTTL time.Duration) {
	const responseCached = "first response"
	const responseNotCached = "subsequent response"
	var testCacheExpiry bool = respTTL > 0
	var respTTLWithBuffer time.Duration = respTTL + (respTTL / 4)
	var requestsExpectedCount int

	requestsReceivedCount := 0
	switch testCacheExpiry {
	case true:
		requestsExpectedCount = 2
	case false:
		requestsExpectedCount = 1
	}

	url := fmt.Sprintf("https://%s/%s", *edgeHost, NewUUID())
	req, _ := http.NewRequest("GET", url, nil)

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

	for requestCount := 0; requestCount < 3; requestCount++ {
		if testCacheExpiry && requestCount == 2 {
			time.Sleep(respTTLWithBuffer)
		}

		resp := RoundTripCheckError(t, req)

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var expectedBody string
		if testCacheExpiry && requestCount > 1 {
			expectedBody = responseNotCached
		} else {
			expectedBody = responseCached
		}

		if receivedBody := string(body); receivedBody != expectedBody {
			t.Errorf(
				"Incorrect response body. Expected %q, got %q",
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

	for _, expectedBody := range responseBodies {
		resp := RoundTripCheckError(t, req)

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if receivedBody := string(body); receivedBody != expectedBody {
			t.Errorf("Incorrect response body. Expected %q, got %q", expectedBody, receivedBody)
		}
	}
}
