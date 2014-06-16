package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

// Should serve a known static error page if all backend servers are down
// and object isn't in cache/stale.
// NB: ideally this should be a page that we control that has a mechanism
//     to alert us that it has been served.
func TestFailoverErrorPageAllServersDown(t *testing.T) {
	t.Error("Not implemented")
}

// Should serve a known static error page if all backend servers return a
// 5xx response and object isn't in cache/stale.
// NB: ideally this should be a page that we control that has a mechanism
//     to alert us that it has been served.
func TestFailoverErrorPageAllServers5xx(t *testing.T) {
	t.Error("Not implemented")
}

// Should back off requests against origin for a very short period of time
// if origin returns a 5xx response so as not to overwhelm it.
func TestFailoverOrigin5xxBackOff(t *testing.T) {
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
	expectedBody := "lucky golden ticket"
	expectedStatus := http.StatusOK
	backendsSawRequest := map[string]bool{}

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		name := originServer.Name
		if !backendsSawRequest[name] {
			w.WriteHeader(http.StatusServiceUnavailable)
			backendsSawRequest[name] = true
		} else {
			t.Errorf("Server %s received more than one request", name)
		}
		w.Write([]byte(name))
	})
	backupServer1.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		name := backupServer1.Name
		if !backendsSawRequest[name] {
			w.Write([]byte(expectedBody))
			backendsSawRequest[name] = true
		} else {
			t.Errorf("Server %s received more than one request", name)
			w.Write([]byte(name))
		}
	})
	backupServer2.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		name := backupServer2.Name
		t.Errorf("Server %s received a request and it shouldn't have", name)
		w.Write([]byte(name))
	})

	url := fmt.Sprintf("https://%s/%s", *edgeHost, NewUUID())
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.RoundTrip(req)

	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != expectedStatus {
		t.Errorf(
			"Received incorrect status code. Expected %d, got %d",
			expectedStatus,
			resp.StatusCode,
		)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if bodyStr := string(body); bodyStr != expectedBody {
		t.Errorf(
			"Received incorrect response body. Expected %q, got %q",
			expectedBody,
			bodyStr,
		)
	}
}

// Should fallback to second mirror if both origin and first mirror are
// down.
func TestFailoverOriginDownFirstMirrorDownUseSecondMirror(t *testing.T) {
	t.Error("Not implemented")
}

// Should fallback to second mirror if both origin and first mirror return
// 5xx responses.
func TestFailoverOrigin5xxFirstMirror5xxUseSecondMirror(t *testing.T) {
	expectedBody := "lucky golden ticket"
	expectedStatus := http.StatusOK
	backendsSawRequest := map[string]bool{}

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		name := originServer.Name
		if !backendsSawRequest[name] {
			w.WriteHeader(http.StatusServiceUnavailable)
			backendsSawRequest[name] = true
		} else {
			t.Errorf("Server %s received more than one request", name)
		}
		w.Write([]byte(name))
	})
	backupServer1.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		name := backupServer1.Name
		if !backendsSawRequest[name] {
			w.WriteHeader(http.StatusServiceUnavailable)
			backendsSawRequest[name] = true
		} else {
			t.Errorf("Server %s received more than one request", name)
		}
		w.Write([]byte(name))
	})
	backupServer2.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		name := backupServer2.Name
		if !backendsSawRequest[name] {
			w.Write([]byte(expectedBody))
			backendsSawRequest[name] = true
		} else {
			t.Errorf("Server %s received more than one request", name)
			w.Write([]byte(name))
		}
	})

	url := fmt.Sprintf("https://%s/%s", *edgeHost, NewUUID())
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.RoundTrip(req)

	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != expectedStatus {
		t.Errorf(
			"Received incorrect status code. Expected %d, got %d",
			expectedStatus,
			resp.StatusCode,
		)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if bodyStr := string(body); bodyStr != expectedBody {
		t.Errorf(
			"Received incorrect response body. Expected %q, got %q",
			expectedBody,
			bodyStr,
		)
	}
}

// Should not fallback to mirror if origin returns a 5xx response with a
// No-Fallback header.
func TestFailoverNoFallbackHeader(t *testing.T) {
	t.Error("Not implemented")
}
