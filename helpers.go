package main

import (
	"fmt"
	"net/http"
	"testing"
)

// HTTP ServeMux with an updateable handler so that tests can pass their own
// anonymous functions in to handle requests.
type CDNServeMux struct {
	Port    int
	handler func(w http.ResponseWriter, r *http.Request)
}

func (s *CDNServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler(w, r)
}

func (s *CDNServeMux) SwitchHandler(h func(w http.ResponseWriter, r *http.Request)) {
	s.handler = h
}

// Start a new server and return the CDNServeMux used.
func StartServer(port int) *CDNServeMux {
	handler := func(w http.ResponseWriter, r *http.Request) {}
	mux := &CDNServeMux{port, handler}
	addr := fmt.Sprintf(":%d", port)
	go http.ListenAndServe(addr, mux)

	return mux
}

// CDNServeMux helper should be ready to serve requests when test suite starts
// and then serve custom handlers each with their own status code.
func testHelpersCDNServeMux(t *testing.T, mux *CDNServeMux) {
	url := fmt.Sprintf("http://localhost:%d/foo", mux.Port)
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatal("Initial probe request failed")
	}

	for i := 300; i < 600; i += 100 {
		mux.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(i)
		})

		resp, err := client.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != i {
			t.Fatal("Request not served by correct handler")
		}
	}
}
