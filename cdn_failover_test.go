package main

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

// checkForSkipFailover skips the calling test if the skipFailover flag has
// been set.
func checkForSkipFailover(t *testing.T) {
	if *skipFailover {
		t.Skip("Failover tests disabled")
	}
}

// Should serve a known static error page if all backend servers are down
// and object isn't in cache/stale.
// NB: ideally this should be a page that we control that has a mechanism
//     to alert us that it has been served.
func TestFailoverErrorPageAllServersDown(t *testing.T) {
	checkForSkipFailover(t)
	ResetBackends(backendsByPriority)

	const expectedStatusCode = http.StatusServiceUnavailable
	var expectedBody string

	switch {
	case vendorFastly:
		expectedBody = "Sorry! We're having issues right now. Please try again later."
	default:
		expectedBody = "Guru Meditation"
	}

	originServer.Stop()
	backupServer1.Stop()
	backupServer2.Stop()

	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatusCode {
		t.Errorf(
			"Invalid StatusCode received. Expected %d, got %d",
			expectedStatusCode,
			resp.StatusCode,
		)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if bodyStr := string(body); !strings.Contains(bodyStr, expectedBody) {
		t.Errorf(
			"Received incorrect response body. Expected to contain %q, got %q",
			expectedBody,
			bodyStr,
		)
	}
}

// Should return the 5xx response from the last backup server if all
// preceeding servers also return a 5xx response.
func TestFailoverErrorPageAllServers5xx(t *testing.T) {
	checkForSkipFailover(t)
	ResetBackends(backendsByPriority)

	const expectedStatusCode = http.StatusServiceUnavailable
	const expectedBody = "lucky golden ticket"

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(originServer.Name))
	})
	backupServer1.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(backupServer1.Name))
	})
	backupServer2.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(expectedBody))
	})

	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatusCode {
		t.Errorf(
			"Invalid StatusCode received. Expected %d, got %d",
			expectedStatusCode,
			resp.StatusCode,
		)
	}

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

// Should back off requests against origin for a very short period of time
// (so as not to overwhelm it) if origin returns a 5xx response.
func TestFailoverOrigin5xxBackOff(t *testing.T) {
	checkForSkipFailover(t)
	ResetBackends(backendsByPriority)

	const expectedBody = "lucky golden ticket"
	const expectedStatus = http.StatusOK

	backupServer1.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expectedBody))
	})
	backupServer2.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		name := backupServer2.Name
		t.Errorf("Server %s received request and it shouldn't have", name)
		w.Write([]byte(name))
	})

	req := NewUniqueEdgeGET(t)

	for requestCount := 1; requestCount < 21; requestCount++ {
		switch requestCount {
		case 1: // Request 1 hits origin but is served from mirror1.
			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(originServer.Name))
			})
		case 2: // Requests 2+ are served directly from mirror1.
			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				name := originServer.Name
				t.Errorf("Server %s received request and it shouldn't have", name)
				w.Write([]byte(name))
			})
		}

		resp := RoundTripCheckError(t, req)
		defer resp.Body.Close()

		if resp.StatusCode != expectedStatus {
			t.Errorf(
				"Request %d received incorrect status code. Expected %d, got %d",
				requestCount,
				expectedStatus,
				resp.StatusCode,
			)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if bodyStr := string(body); bodyStr != expectedBody {
			t.Errorf(
				"Request %d received incorrect response body. Expected %q, got %q",
				requestCount,
				expectedBody,
				bodyStr,
			)
		}
	}
}

// Should fallback to first mirror if origin is down and object is not in
// cache (active or stale).
func TestFailoverOriginDownUseFirstMirror(t *testing.T) {
	checkForSkipFailover(t)
	ResetBackends(backendsByPriority)

	expectedBody := "lucky golden ticket"
	expectedStatus := http.StatusOK

	originServer.Stop()
	backupServer1.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expectedBody))
	})
	backupServer2.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		name := backupServer2.Name
		t.Errorf("Server %s received a request and it shouldn't have", name)
		w.Write([]byte(name))
	})

	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Errorf(
			"Received incorrect status code. Expected %d, got %d",
			expectedStatus,
			resp.StatusCode,
		)
	}

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

// Should fallback to first mirror if origin returns 5xx response and object
// is not in cache (active or stale).
func TestFailoverOrigin5xxUseFirstMirror(t *testing.T) {
	checkForSkipFailover(t)
	ResetBackends(backendsByPriority)

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

	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Errorf(
			"Received incorrect status code. Expected %d, got %d",
			expectedStatus,
			resp.StatusCode,
		)
	}

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
	checkForSkipFailover(t)
	ResetBackends(backendsByPriority)

	expectedBody := "lucky golden ticket"
	expectedStatus := http.StatusOK

	originServer.Stop()
	backupServer1.Stop()
	backupServer2.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expectedBody))
	})

	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Errorf(
			"Received incorrect status code. Expected %d, got %d",
			expectedStatus,
			resp.StatusCode,
		)
	}

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

// Should fallback to second mirror if both origin and first mirror return
// 5xx responses.
func TestFailoverOrigin5xxFirstMirror5xxUseSecondMirror(t *testing.T) {
	checkForSkipFailover(t)
	ResetBackends(backendsByPriority)

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

	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Errorf(
			"Received incorrect status code. Expected %d, got %d",
			expectedStatus,
			resp.StatusCode,
		)
	}

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
// No-Fallback header. In order to allow applications to present their own
// error pages.
func TestFailoverNoFallbackHeader(t *testing.T) {
	checkForSkipFailover(t)
	ResetBackends(backendsByPriority)

	const headerName = "No-Fallback"
	const expectedStatus = http.StatusServiceUnavailable
	const expectedBody = "custom error page"

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerName, "")
		w.WriteHeader(expectedStatus)
		w.Write([]byte(expectedBody))
	})
	backupServer1.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		name := backupServer1.Name
		t.Errorf("Server %s received request and it shouldn't have", name)
		w.Write([]byte(name))
	})
	backupServer2.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		name := backupServer2.Name
		t.Errorf("Server %s received request and it shouldn't have", name)
		w.Write([]byte(name))
	})

	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Errorf(
			"Received incorrect status code. Expected %d, got %d",
			expectedStatus,
			resp.StatusCode,
		)
	}

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
