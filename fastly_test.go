package main

import (
	"fmt"
	"net/http"
	"testing"
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

	t.Logf("req %s X-Served-By: %s",
		ident, resp.Header.Get("X-Served-By"))

	if status := resp.StatusCode; status != expectedStatus {
		t.Errorf("req %s wrong status code. Expected %d, got %d",
			ident, status, expectedStatus)
	}
}
