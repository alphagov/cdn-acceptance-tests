package main

import (
	"flag"
	"testing"
)

var edgeHostName = flag.String("edge", "www.gov.uk", "Hostname of edge")

// Should redirect from HTTP to HTTPS without hitting origin.
func TestProtocolRedirect(t *testing.T) {
	t.Fatal("Not yet implemented")
}
