package main

import (
	"flag"
	"fmt"
	"testing"
	"net/http"
)

var edgeHostName = flag.String("edge", "www.gov.uk", "Hostname of edge")

// Should redirect from HTTP to HTTPS without hitting origin.
func TestProtocolRedirect(t *testing.T) {
	sourceUrl := fmt.Sprintf("http://%s/", *edgeHostName)
	destUrl := fmt.Sprintf("https://%s/", *edgeHostName)

	client := &http.Transport{}
	req, _ := http.NewRequest("GET", sourceUrl, nil)
	resp, err := client.RoundTrip(req)

	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != 301 {
		t.Errorf("Status code expected 301, got %s", resp.StatusCode)
	}
	if d := resp.Header.Get("Location"); d != destUrl {
		t.Errorf("Location header expected %s, got %s", destUrl, d)
	}

	t.Error("Not implemented test to confirm that it doesn't hit origin")
}
