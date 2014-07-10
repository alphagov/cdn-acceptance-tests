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
	edgeHost          = flag.String("edgeHost", "", "Hostname of edge")
	originPort        = flag.Int("originPort", 8080, "Origin port to listen on for requests")
	backupPort1       = flag.Int("backupPort1", 8081, "Backup1 port to listen on for requests")
	backupPort2       = flag.Int("backupPort2", 8082, "Backup2 port to listen on for requests")
	skipVerifyTLS     = flag.Bool("skipVerifyTLS", false, "Skip TLS cert verification if set")
	disableBackendTLS = flag.Bool("disableBackendTLS", false, "Disable TLS on backends")
)

// These consts and vars are available to all tests.
const requestTimeout = time.Second * 5

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

	if *edgeHost == "" {
		fmt.Printf("ERROR: -edgeHost must be set to the CDN edge hostname we wish to test against\n\n")
		fmt.Println("Usage:")
		flag.PrintDefaults()
		os.Exit(1)
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

	originServer = &CDNBackendServer{
		Name:        "origin",
		Port:        *originPort,
		TLSDisabled: *disableBackendTLS,
	}
	backupServer1 = &CDNBackendServer{
		Name:        "backup1",
		Port:        *backupPort1,
		TLSDisabled: *disableBackendTLS,
	}
	backupServer2 = &CDNBackendServer{
		Name:        "backup2",
		Port:        *backupPort2,
		TLSDisabled: *disableBackendTLS,
	}

	backendsByPriority = []*CDNBackendServer{
		originServer,
		backupServer1,
		backupServer2,
	}

	log.Println("Confirming that CDN is healthy")
	ResetBackends(backendsByPriority)

}

// CacheHostIpAddress looks up the IP address for a given host name,
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
// http.Transport. It uses the IP address returned by CachedHostIpAddresss
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
