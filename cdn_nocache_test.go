package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

// Should send request to origin by default
func TestNoCacheNewRequestOrigin(t *testing.T) {
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

// Should not cache the response to a POST request.
func TestNoCachePOST(t *testing.T) {
	requestsReceivedCount := 0
	responseBodies := []string{
		"first response",
		"second response",
		"third response",
	}

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(responseBodies[requestsReceivedCount]))
		requestsReceivedCount++
	})

	url := fmt.Sprintf("https://%s/%s", *edgeHost, NewUUID())
	req, _ := http.NewRequest("POST", url, nil)

	for _, expectedBody := range responseBodies {
		resp, err := client.RoundTrip(req)

		if err != nil {
			t.Fatal(err)
		}

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

// Should not cache the response to a request with a `Authorization` header.
func TestNoCacheHeaderAuthorization(t *testing.T) {
	t.Error("Not implemented")
}

// Should not cache the response to a request with a `Cookie` header.
func TestNoCacheHeaderCookie(t *testing.T) {
	t.Error("Not implemented")
}

// Should not cache a response with a `Set-Cookie` header.
func TestNoCacheHeaderSetCookie(t *testing.T) {
	requestsReceivedCount := 0
	responseBodies := []string{
		"first response",
		"second response",
		"third response",
	}

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "sekret=mekmitasdigoat")
		w.Write([]byte(responseBodies[requestsReceivedCount]))
		requestsReceivedCount++
	})

	url := fmt.Sprintf("https://%s/%s", *edgeHost, NewUUID())
	req, _ := http.NewRequest("GET", url, nil)

	for _, expectedBody := range responseBodies {
		resp, err := client.RoundTrip(req)

		if err != nil {
			t.Fatal(err)
		}

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

// Should not cache a response with a `Cache-Control: private` header.
func TestNoCacheHeaderCacheControlPrivate(t *testing.T) {
	t.Error("Not implemented")
}
