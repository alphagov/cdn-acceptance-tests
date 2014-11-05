package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
	"strconv"
)

func TestFastlyUpDown(t *testing.T) {
	for i := 1; i <= 20; i++ {
		ResetBackends(backendsByPriority)

		testFastlyReq(t, fmt.Sprintf("%d:up", i), http.StatusOK)
		stopBackends(backendsByPriority)
		testFastlyReq(t, fmt.Sprintf("%d:down", i), http.StatusServiceUnavailable)
	}
}

func testFastlyReq(t *testing.T, ident string, expectedStatus int) {
	req := NewUniqueEdgeGET(t)
	resp := RoundTripCheckError(t, req)

	for _, backend := range backendsByPriority {
		backend.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("req %s received by %s: %s",
				backend.Name,
				strconv.FormatInt(time.Now().UnixNano(), 10),
			)
		})
	}

	t.Logf("req %s X-Served-By: %s",
		ident, resp.Header.Get("X-Served-By"))

	if status := resp.StatusCode; status != expectedStatus {
		t.Errorf("req %s wrong status code. Expected %d, got %d",
			ident, status, expectedStatus)
	}
}
