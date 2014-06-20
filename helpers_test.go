package main

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
)

// CDNServeMux helper should be ready to serve requests when test suite starts
// and then serve custom handlers each with their own status code.
func TestHelpersCDNServeMuxHandlers(t *testing.T) {
	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {})

	url := fmt.Sprintf("http://localhost:%d/foo", originServer.Port)
	req, _ := http.NewRequest("GET", url, nil)
	resp := RoundTripCheckError(t, req)

	if resp.StatusCode != 200 {
		t.Error("First request to default handler failed")
	}

	for _, statusCode := range []int{301, 302, 403, 404} {
		originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
		})

		resp := RoundTripCheckError(t, req)
		if resp.StatusCode != statusCode {
			t.Errorf("SwitchHandler didn't work. Got %d, expected %d", resp.StatusCode, statusCode)
		}
	}
}

// CDNServeMux should always respond to HEAD requests in order for the CDN to
// determine the health of our origin.
func TestHelpersCDNServeMuxProbes(t *testing.T) {
	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		t.Error("HEAD request incorrectly served by CDNServeMux.handler")
	})

	url := fmt.Sprintf("http://localhost:%d/", originServer.Port)
	req, _ := http.NewRequest("HEAD", url, nil)
	resp := RoundTripCheckError(t, req)

	if resp.StatusCode != 200 || resp.Header.Get("PING") != "PONG" {
		t.Error("HEAD request for '/' served incorrectly")
	}
}

func TestHelpersCDNServeStop(t *testing.T) {
	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {})

	url := fmt.Sprintf("http://localhost:%d/foo", originServer.Port)
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Error("originServer should be up and responding, prior to Stop operation")
	}

	originServer.Stop()

	resp, err = client.RoundTrip(req)
	if err == nil {
		t.Error("Client connection succeeded. The server should be refusing requests by now.")
	}

	re := regexp.MustCompile(`EOF`)
	if !re.MatchString(fmt.Sprintf("%s", err)) {
		t.Errorf("Connection error %q is not as expected", err)
	}

	// Reset back to a known-good state
	backupServer1.Stop()
	backupServer2.Stop()
	StartBackendsInOrder(*edgeHost)

}
