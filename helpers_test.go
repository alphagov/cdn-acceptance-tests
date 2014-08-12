package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// CDNBackendServer instance should be ready to serve requests when test
// suite starts and then serve custom handlers each with their own status
// code.
func TestHelpersCDNBackendServerHandlers(t *testing.T) {
	ResetBackends(backendsByPriority)

	url := originServer.server.URL + "/" + NewUUID()
	req, _ := http.NewRequest("GET", url, nil)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Error("First request to default handler failed")
	}

	for _, statusCode := range []int{301, 302, 403, 404} {
		originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
		})

		resp := RoundTripCheckError(t, req)
		defer resp.Body.Close()

		if resp.StatusCode != statusCode {
			t.Errorf("SwitchHandler didn't work. Got %d, expected %d", resp.StatusCode, statusCode)
		}
	}
}

// CDNBackendServer should always respond to HEAD requests in order for the
// CDN to determine the health of our origin.
func TestHelpersCDNBackendServerProbes(t *testing.T) {
	ResetBackends(backendsByPriority)

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		t.Error("HEAD request incorrectly served by CDNBackendServer.handler")
	})

	url := originServer.server.URL + "/"
	req, _ := http.NewRequest("HEAD", url, nil)
	resp := RoundTripCheckError(t, req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 || resp.Header.Get("PING") != "PONG" {
		t.Error("HEAD request for '/' served incorrectly")
	}
}

func TestHelpersCDNServeStop(t *testing.T) {
	ResetBackends(backendsByPriority)

	var connectionErrorRegex *regexp.Regexp = regexp.MustCompile(`(^EOF| connection refused)$`)
	var expectedStarted bool

	expectedStarted = true
	if started := originServer.IsStarted(); started != expectedStarted {
		t.Errorf(
			"originServer.IsStarted() incorrect. Expected %t, got %t",
			expectedStarted,
			started,
		)
	}

	url := originServer.server.URL + "/" + NewUUID()
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Error("originServer should be up and responding, prior to Stop operation")
	}

	originServer.Stop()
	expectedStarted = false
	if started := originServer.IsStarted(); started {
		t.Errorf(
			"originServer.IsStarted() incorrect. Expected %t, got %t",
			expectedStarted,
			started,
		)
	}

	resp, err = client.RoundTrip(req)
	if err == nil {
		defer resp.Body.Close()
		t.Error("Client connection succeeded. The server should be refusing requests by now.")
	}

	if !connectionErrorRegex.MatchString(fmt.Sprintf("%s", err)) {
		t.Errorf("Connection error %q is not as expected", err)
	}
}

// CDNBackendServer should assign a random port when started for the first
// time with a port of 0. Subsequent starts should retain the assigned port
// from the first start.
func TestHelpersCDNBackendServerRandomPort(t *testing.T) {
	const initialPort = 0
	var assignedPort int

	backend := CDNBackendServer{
		Name: "test",
		Port: initialPort,
	}

	backend.Start()
	defer backend.Stop()

	assignedPort = backend.Port
	if assignedPort == initialPort {
		t.Errorf(
			"Expected backend port != %d, got %d",
			initialPort,
			assignedPort,
		)
	}

	backend.Stop()
	backend.Start()

	if backend.Port != assignedPort {
		t.Errorf(
			"Expected backend port == %d, got %d",
			backend.Port,
			assignedPort,
		)
	}
}

// CDNBackendServer should use TLS by default as evidenced by an HTTPS URL
// from `httptest.Server`.
func TestHelpersCDNBackendServerTLSEnabled(t *testing.T) {
	const expectedURLPrefix = "https://"

	backend := CDNBackendServer{
		Name: "test",
		Port: 0,
	}

	backend.Start()
	defer backend.Stop()

	if url := backend.server.URL; !strings.HasPrefix(url, expectedURLPrefix) {
		t.Errorf(
			"Expected backend URL to begin with %q, got %q",
			expectedURLPrefix,
			url,
		)
	}
}

// CDNBackendServer should use a self-signed certificate from
// `httptest.Server` if `TLSCerts` is empty (default).
func TestHelpersCDNBackendServerTLSDefaultCert(t *testing.T) {
	expectedCertDNSNames := []string{"example.com"}
	expectedCertIPAddresses := []net.IP{
		net.IPv4(127, 0, 0, 1).To4(),
		net.IPv6loopback,
	}

	backend := CDNBackendServer{
		Name: "test",
		Port: 0,
	}

	backend.Start()
	defer backend.Stop()

	conn, err := tls.Dial(
		"tcp",
		backend.server.Listener.Addr().String(),
		&tls.Config{
			InsecureSkipVerify: true,
		},
	)
	if err != nil {
		t.Fatal("Error connecting: ", err)
	}

	cert := conn.ConnectionState().PeerCertificates[0]
	if !reflect.DeepEqual(cert.DNSNames, expectedCertDNSNames) {
		t.Errorf(
			"Incorrect cert SAN DNSNames. Expected %q, got %q",
			expectedCertDNSNames,
			cert.DNSNames,
		)
	}
	if !reflect.DeepEqual(cert.IPAddresses, expectedCertIPAddresses) {
		t.Errorf(
			"Incorrect cert SAN IPAddresses. Expected %q, got %q",
			expectedCertIPAddresses,
			cert.IPAddresses,
		)
	}
}

// CDNBackendServer should use custom certificate and key if `TLSCerts` is
// passed.
func TestHelpersCDNBackendServerTLSCustomCert(t *testing.T) {
	expectedCertDNSNames := []string{"cdn-acceptance-tests.example.com"}
	expectedCertIPAddresses := []net.IP{
		net.IPv4(203, 0, 113, 10).To4(),
	}

	customCertKey, err := tls.X509KeyPair(customCert, customKey)
	if err != nil {
		t.Fatal(err)
	}

	backend := CDNBackendServer{
		Name:     "test",
		Port:     0,
		TLSCerts: []tls.Certificate{customCertKey},
	}

	backend.Start()
	defer backend.Stop()

	conn, err := tls.Dial(
		"tcp",
		backend.server.Listener.Addr().String(),
		&tls.Config{
			InsecureSkipVerify: true,
		},
	)
	if err != nil {
		t.Fatal("Error connecting: ", err)
	}

	cert := conn.ConnectionState().PeerCertificates[0]
	if !reflect.DeepEqual(cert.DNSNames, expectedCertDNSNames) {
		t.Errorf(
			"Incorrect cert SAN DNSNames. Expected %q, got %q",
			expectedCertDNSNames,
			cert.DNSNames,
		)
	}
	if !reflect.DeepEqual(cert.IPAddresses, expectedCertIPAddresses) {
		t.Errorf(
			"Incorrect cert SAN IPAddresses. Expected %q, got %q",
			expectedCertIPAddresses,
			cert.IPAddresses,
		)
	}
}

// generated from src/pkg/crypto/tls:
// go run generate_cert.go --rsa-bits 512 --host 203.0.113.10,cdn-acceptance-tests.example.com --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
var customCert = []byte(`-----BEGIN CERTIFICATE-----
MIIBfDCCASigAwIBAgIBADALBgkqhkiG9w0BAQUwEjEQMA4GA1UEChMHQWNtZSBD
bzAeFw03MDAxMDEwMDAwMDBaFw00OTEyMzEyMzU5NTlaMBIxEDAOBgNVBAoTB0Fj
bWUgQ28wXDANBgkqhkiG9w0BAQEFAANLADBIAkEArfMXU/ttiLo1JIPbsprMHNmE
DazpOAudumBLGjzgiUVrsfgH2oYlfNweinSzPYF90B2yQf/zZVLS/0x3ZKajNwID
AQABo2swaTAOBgNVHQ8BAf8EBAMCAKQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYD
VR0TAQH/BAUwAwEB/zAxBgNVHREEKjAogiBjZG4tYWNjZXB0YW5jZS10ZXN0cy5l
eGFtcGxlLmNvbYcEywBxCjALBgkqhkiG9w0BAQUDQQBe5lFZYSf7OAe97BkT/BKo
Ewqkmup2sFljHPXeS1ZRTbgJSjyOnskqf4psHEbyc2YjjYtXbsid7XjM3AyfB93F
-----END CERTIFICATE-----`)
var customKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIBOQIBAAJBAK3zF1P7bYi6NSSD27KazBzZhA2s6TgLnbpgSxo84IlFa7H4B9qG
JXzcHop0sz2BfdAdskH/82VS0v9Md2SmozcCAwEAAQJAKto1CAJrpIBC8UDukZxi
5kSLrJbJSX5LGAv61Hbk1cv1U6eiqPo7VkkKgjiJH5kpcwzdA1dHSuM/bhk+iqQ1
CQIhANbZfMSxw1bDH/B4vvXD9ysjdjnlSwgVHlKnRzVT7sPbAiEAz0QxpwMy+FMD
D5bWtz25z9MqB6sMTyBHXYSU+C0/atUCIH+kxtO1KPCrDJa5pfotavNeJidParxq
j5FbgJrWOsxxAiBb550stVpwij6dNwFWl2RBJx1H8SywGVwLt7JmqYmpUQIgf0HJ
YrI972WOb4pQEuKgIKMuJ/tHa99iMcmmUjbCNSI=
-----END RSA PRIVATE KEY-----`)
