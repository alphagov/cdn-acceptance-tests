package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

// Should cache first response and return it on second request without
// hitting origin again.
func TestCacheFirstResponse(t *testing.T) {
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

// Should set a default TTL if the response doesn't set one.
func TestCacheDefaultTTL(t *testing.T) {
	t.Error("Not implemented")
}
