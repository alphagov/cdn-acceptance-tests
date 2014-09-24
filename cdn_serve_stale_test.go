package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

// Should serve reponse from first mirror and replace stale object if origin
// is down and health check has *not* expired.
// FIXME: This is not desired behaviour. We should serve from stale
//        immediately and not replace the stale object in cache.
func TestServeStaleOriginDownHealthCheckNotExpiredReplace(t *testing.T) {
	checkForSkipFailover(t)
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
func TestServeStaleOriginDownHealthCheckHasExpired(t *testing.T) {
	checkForSkipFailover(t)
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
func TestServeStaleOrigin5xx(t *testing.T) {
	checkForSkipFailover(t)
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
