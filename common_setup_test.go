package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
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

var hardCachedEdgeHostIp string

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
		Dial:                  HardCachedHostDial,
	}
	originServer = StartServer(*originPort)

	log.Println("Confirming that CDN is healthy")
	err := confirmEdgeIsHealthy(originServer, *edgeHost)
	if err != nil {
		log.Fatal(err)
	}
}

func CachedHostIpAddress(host string) string {
	if hardCachedEdgeHostIp == "" {
		ipAddresses, err := net.LookupHost(host)
		if err != nil {
			log.Fatal(err)
		}
		hardCachedEdgeHostIp = ipAddresses[0]
	}
	return hardCachedEdgeHostIp
}

func HardCachedHostDial(network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Fatal(err)
	}
	if host == "localhost" {
		return net.Dial(network, addr)
	}
	ipAddr := CachedHostIpAddress(host)
	return net.Dial(network, fmt.Sprintf("%s:%s", ipAddr, port))
}
