package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

var (
	backendCert   = flag.String("backendCert", "", "Override self-signed cert for backend TLS")
	backendKey    = flag.String("backendKey", "", "Override self-signed cert, must be provided with -backendCert")
	backupPort1   = flag.Int("backupPort1", 8081, "Backup1 port to listen on for requests")
	backupPort2   = flag.Int("backupPort2", 8082, "Backup2 port to listen on for requests")
	edgeHost      = flag.String("edgeHost", "", "Hostname of edge")
	originPort    = flag.Int("originPort", 8080, "Origin port to listen on for requests")
	skipFailover  = flag.Bool("skipFailover", false, "Skip failover tests and only setup the origin backend")
	skipVerifyTLS = flag.Bool("skipVerifyTLS", false, "Skip TLS cert verification if set")
	usage         = flag.Bool("usage", false, "Print usage")
	vendor        = flag.String("vendor", "", "Name of vendor; run tests specific to vendor")
	// This only works with tests that use RoundTripCheckError(), that either
	// are either failing or run with the -v flag.
	debugResp = flag.Bool("debugResp", false, "Log responses for debugging")
)

var (
	vendorFastly     bool = false
	vendorCloudflare bool = false
)

// These consts and vars are available to all tests.
const requestTimeout = time.Second * 5
const requestSlowThreshold = time.Second
const notImplementedForVendor = "Test not yet implemented for your selected vendor or no vendor specified"
const notSupportedByVendor = "Feature not supported by your selected vendor"

var (
	client             *http.Transport
	originServer       *CDNBackendServer
	backupServer1      *CDNBackendServer
	backupServer2      *CDNBackendServer
	backendsByPriority []*CDNBackendServer
)

var hardCachedEdgeHostIp string

// Setup clients and servers.
func init() {

	flag.Parse()

	if *usage {
		flag.Usage()
		os.Exit(0)
	}

	if *edgeHost == "" {
		fmt.Printf("ERROR: -edgeHost must be set to the CDN edge hostname we wish to test against\n\n")
		flag.Usage()
		os.Exit(1)
	}

	switch *vendor {
	case "cloudflare":
		vendorCloudflare = true
	case "fastly":
		vendorFastly = true
	case "":
		log.Println("No vendor specified")
	default:
		log.Fatalf("Vendor %q unrecognised; aborting", *vendor)
	}

	tlsOptions := &tls.Config{}
	if *skipVerifyTLS {
		tlsOptions.InsecureSkipVerify = true
	}
	client = &http.Transport{
		ResponseHeaderTimeout: requestTimeout,
		TLSClientConfig:       tlsOptions,
		Dial:                  HardCachedHostDial,
	}

	var backendCerts []tls.Certificate
	if *backendCert != "" || *backendKey != "" {
		var err error
		backendCerts = make([]tls.Certificate, 1)
		backendCerts[0], err = tls.LoadX509KeyPair(*backendCert, *backendKey)

		if err != nil {
			log.Fatal(err)
		}
	}

	originServer = &CDNBackendServer{
		Name:     "origin",
		Port:     *originPort,
		TLSCerts: backendCerts,
	}
	backendsByPriority = []*CDNBackendServer{
		originServer,
	}

	if !*skipFailover {
		backupServer1 = &CDNBackendServer{
			Name:     "backup1",
			Port:     *backupPort1,
			TLSCerts: backendCerts,
		}
		backupServer2 = &CDNBackendServer{
			Name:     "backup2",
			Port:     *backupPort2,
			TLSCerts: backendCerts,
		}
		backendsByPriority = append(
			backendsByPriority,
			backupServer1,
			backupServer2,
		)
	}

	log.Println("Confirming that CDN is healthy")
	ResetBackends(backendsByPriority)

}

// CachedHostIpAddress looks up the IP address for a given host name,
// and caches the first IP address returned. Subsequent requests always
// return this address, preventing further DNS requests.
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

// HardCachedHostDial acts as a replacement Dial function, ostensibly for
// http.Transport. It uses the IP address returned by CachedHostIpAddress
// and passes that to the stock net.Dial function, to prevent repeat DNS
// lookups of the provided hostname in addr. This is to prevent us from switching
// from one CDN location to another mid-test.
func HardCachedHostDial(network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Fatal(err)
	}
	if host != *edgeHost {
		return net.Dial(network, addr)
	}
	ipAddr := CachedHostIpAddress(host)
	return net.Dial(network, net.JoinHostPort(ipAddr, port))
}
