package main

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"
)

// HTTP ServeMux with an updateable handler so that tests can pass their own
// anonymous functions in to handle requests.
type CDNServeMux struct {
	Port    int
	handler func(w http.ResponseWriter, r *http.Request)
}

func (s *CDNServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "HEAD" && r.URL.Path == "/" {
		w.Header().Set("PING", "PONG")
		return
	}

	s.handler(w, r)
}

func (s *CDNServeMux) SwitchHandler(h func(w http.ResponseWriter, r *http.Request)) {
	s.handler = h
}

// Start a new server and return the CDNServeMux used.
func StartServer(port int) *CDNServeMux {
	handler := func(w http.ResponseWriter, r *http.Request) {}
	mux := &CDNServeMux{port, handler}
	addr := fmt.Sprintf(":%d", port)

	go func() {
		err := http.ListenAndServe(addr, mux)
		if err != nil {
			panic(err)
		}
	}()

	return mux
}

// Return a v4 (random) UUID string.
// This might not be strictly RFC4122 compliant, but it will do. Credit:
// https://groups.google.com/d/msg/golang-nuts/Rn13T6BZpgE/dBaYVJ4hB5gJ
func NewUUID() string {
	bs := make([]byte, 16)
	rand.Read(bs)
	bs[6] = (bs[6] & 0x0f) | 0x40
	bs[8] = (bs[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", bs[0:4], bs[4:6], bs[6:8], bs[8:10], bs[10:])
}

// Confirm that the edge (CDN) is working correctly. This may take some time
// because our CDNServeMux needs to receive and respond to enough probe
// health checks to be considered up.
func confirmEdgeIsHealthy(mux *CDNServeMux, edgeHost string) error {
	const maxRetries = 20
	const timeBetweenAttempts = time.Duration(2 * time.Second)
	const waitForCdnProbeToPropagate = time.Duration(5 * time.Second)

	mux.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	var sourceUrl string

	for try := 0; try <= maxRetries; try++ {
		uuid := NewUUID()
		sourceUrl = fmt.Sprintf("https://%s/?cacheBuster=%s", edgeHost, uuid)
		req, _ := http.NewRequest("GET", sourceUrl, nil)
		resp, err := client.RoundTrip(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == 200 {
			if try != 0 {
				time.Sleep(waitForCdnProbeToPropagate)
			}
			return nil // all is well!
		}
		time.Sleep(timeBetweenAttempts)
	}
	return fmt.Errorf("CDN still not available after %d attempts", maxRetries)

}
