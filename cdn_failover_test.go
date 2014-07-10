package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Should serve a known static error page if all backend servers are down
// and object isn't in cache/stale.
// NB: ideally this should be a page that we control that has a mechanism
//     to alert us that it has been served.
func TestFailoverErrorPageAllServersDown(t *testing.T) {
	ResetBackends(backendsByPriority)

	const expectedStatusCode = http.StatusServiceUnavailable
	const expectedBody = "Guru Meditation"

	originServer.Stop()
	backupServer1.Stop()
	backupServer2.Stop()

	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)

	if resp.StatusCode != expectedStatusCode {
		t.Errorf(
			"Invalid StatusCode received. Expected %d, got %d",
			expectedStatusCode,
			resp.StatusCode,
		)
	}

	defer resp.Body.Close()
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

	if resp.StatusCode != expectedStatusCode {
		t.Errorf(
			"Invalid StatusCode received. Expected %d, got %d",
			expectedStatusCode,
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

// Should back off requests against origin for a very short period of time
// (so as not to overwhelm it) if origin returns a 5xx response.
func TestFailoverOrigin5xxBackOff(t *testing.T) {
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

		if resp.StatusCode != expectedStatus {
			t.Errorf(
				"Request %d received incorrect status code. Expected %d, got %d",
				requestCount,
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
				"Request %d received incorrect response body. Expected %q, got %q",
				requestCount,
				expectedBody,
				bodyStr,
			)
		}
	}
}

// Should serve reponse from first mirror and replace stale object if origin
// is down and health check has *not* expired.
// FIXME: This is not desired behaviour. We should serve from stale
//        immediately and not replace the stale object in cache.
func TestFailoverOriginDownHealthCheckNotExpiredReplaceStale(t *testing.T) {
	ResetBackends(backendsByPriority)

	const expectedResponseStale = "going off like stilton"
	const expectedResponseFresh = "as fresh as daisies"

	const respTTL = time.Duration(2 * time.Second)
	const respTTLWithBuffer = 5 * respTTL
	headerValue := fmt.Sprintf("max-age=%.0f", respTTL.Seconds())

	backupServer2.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		name := backupServer2.Name
		t.Errorf("Server %s received request and it shouldn't have", name)
		w.Write([]byte(name))
	})

	req := NewUniqueEdgeGET(t)

	var expectedBody string
	for requestCount := 1; requestCount < 4; requestCount++ {
		switch requestCount {
		case 1: // Request 1 populates cache.
			expectedBody = expectedResponseStale

			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", headerValue)
				w.Write([]byte(expectedBody))
			})
			backupServer1.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				name := backupServer1.Name
				t.Errorf("Server %s received request and it shouldn't have", name)
				w.Write([]byte(name))
			})
		case 2: // Request 2 comes from mirror and invalidates stale.
			time.Sleep(respTTLWithBuffer)
			expectedBody = expectedResponseFresh

			originServer.Stop()
			backupServer1.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(expectedBody))
			})
		case 3: // Request 3 still comes from cache when origin is back.
			expectedBody = expectedResponseFresh

			ResetBackends(backendsByPriority)
			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				name := originServer.Name
				t.Errorf("Server %s received request and it shouldn't have", name)
				w.Write([]byte(name))
			})
			backupServer1.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				name := backupServer1.Name
				t.Errorf("Server %s received request and it shouldn't have", name)
				w.Write([]byte(name))
			})
		}

		resp := RoundTripCheckError(t, req)

		defer resp.Body.Close()
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

// Should serve stale object and not hit mirror(s) if origin is down, health
// check has expired, and object is beyond TTL but still in cache.
// FIXME: This is not quite desired behaviour. We should not have to wait
//				for the stale object to become available.
func TestFailoverOriginDownHealthCheckHasExpiredServeStale(t *testing.T) {
	ResetBackends(backendsByPriority)

	const expectedBody = "going off like stilton"
	// Allow health check to expire. Depends on window/threshold/interval.
	const healthCheckExpire = time.Duration(20 * time.Second)
	const respTTL = time.Duration(2 * time.Second)
	headerValue := fmt.Sprintf("max-age=%.0f", respTTL.Seconds())

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

	for requestCount := 1; requestCount < 3; requestCount++ {
		switch requestCount {
		case 1: // Request 1 populates cache.
			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", headerValue)
				w.Write([]byte(expectedBody))
			})
		case 2: // Request 2 come from stale.
			originServer.Stop()
			time.Sleep(healthCheckExpire)
		}

		resp := RoundTripCheckError(t, req)

		defer resp.Body.Close()
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

// Should serve stale object and not hit mirror(s) if origin returns a 5xx
// response and object is beyond TTL but still in cache.
func TestFailoverOrigin5xxServeStale(t *testing.T) {
	ResetBackends(backendsByPriority)

	const expectedResponseStale = "going off like stilton"
	const expectedResponseFresh = "as fresh as daisies"

	const respTTL = time.Duration(2 * time.Second)
	const respTTLWithBuffer = 5 * respTTL
	// Allow varnish's beresp.saintmode to expire.
	const waitSaintMode = time.Duration(5 * time.Second)
	headerValue := fmt.Sprintf("max-age=%.0f", respTTL.Seconds())

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

	var expectedBody string
	for requestCount := 1; requestCount < 6; requestCount++ {
		switch requestCount {
		case 1: // Request 1 populates cache.
			expectedBody = expectedResponseStale

			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", headerValue)
				w.Write([]byte(expectedBody))
			})
		case 2: // Requests 2,3,4 come from stale.
			time.Sleep(respTTLWithBuffer)
			expectedBody = expectedResponseStale

			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(originServer.Name))
			})
		case 5: // Last request comes directly from origin again.
			time.Sleep(waitSaintMode)
			expectedBody = expectedResponseFresh

			originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(expectedBody))
			})
		}

		resp := RoundTripCheckError(t, req)

		defer resp.Body.Close()
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

// Should fallback to first mirror if origin returns 5xx response and object
// is not in cache (active or stale).
func TestFailoverOrigin5xxUseFirstMirror(t *testing.T) {
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

// Should fallback to second mirror if both origin and first mirror return
// 5xx responses.
func TestFailoverOrigin5xxFirstMirror5xxUseSecondMirror(t *testing.T) {
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
// No-Fallback header. In order to allow applications to present their own
// error pages.
func TestFailoverNoFallbackHeader(t *testing.T) {
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
