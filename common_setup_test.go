package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"time"
)

var (
	edgeHost      = flag.String("edgeHost", "www.gov.uk", "Hostname of edge")
	originPort    = flag.Int("originPort", 8080, "Origin port to listen on for requests")
	skipVerifyTLS = flag.Bool("skipVerifyTLS", false, "Skip TLS cert verification if set")
)

// These consts and vars are available to all tests.
const requestTimeout = time.Second * 5

var (
	client       *http.Transport
	originServer *CDNServeMux
)

// Setup clients and servers.
func init() {

	flag.Parse()

	tlsOptions := &tls.Config{}
	if *skipVerifyTLS {
		tlsOptions.InsecureSkipVerify = true
	}

	client = &http.Transport{
		ResponseHeaderTimeout: requestTimeout,
		TLSClientConfig:       tlsOptions,
	}
	originServer = StartServer(*originPort)

	log.Println("Confirming that CDN is healthy")
	err := confirmEdgeIsHealthy(originServer, *edgeHost)
	if err != nil {
		log.Fatal(err)
	}
}
