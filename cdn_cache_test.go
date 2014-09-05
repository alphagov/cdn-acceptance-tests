package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Should cache first response for an unspecified period of time if when it
// doesn't specify it's own cache headers. Subsequent requests should return
// a cached response.
func TestCacheFirstResponse(t *testing.T) {
	ResetBackends(backendsByPriority)

	testRequestsCachedIndefinite(t, nil)
}

// Should cache responses for the period defined in a `Expires: n` response
// header.
func TestCacheExpires(t *testing.T) {
	ResetBackends(backendsByPriority)

	const cacheDuration = time.Duration(5 * time.Second)

	handler := func(w http.ResponseWriter) {
		headerValue := time.Now().UTC().Add(cacheDuration).Format(http.TimeFormat)
		w.Header().Set("Expires", headerValue)
	}

	testRequestsCachedDuration(t, handler, cacheDuration)
}

// Should cache responses for the period defined in a `Cache-Control:
// max-age=n` response header.
func TestCacheCacheControlMaxAge(t *testing.T) {
	ResetBackends(backendsByPriority)

	const cacheDuration = time.Duration(5 * time.Second)
	headerValue := fmt.Sprintf("max-age=%.0f", cacheDuration.Seconds())

	handler := func(w http.ResponseWriter) {
		w.Header().Set("Cache-Control", headerValue)
	}

	testRequestsCachedDuration(t, handler, cacheDuration)
}

// Should cache responses for the period defined in a `Cache-Control:
// max-age=n` response header when a `Expires: n*2` header is also present.
func TestCacheExpiresAndMaxAge(t *testing.T) {
	ResetBackends(backendsByPriority)

	const cacheDuration = time.Duration(5 * time.Second)
	const expiresDuration = cacheDuration * 2

	maxAgeValue := fmt.Sprintf("max-age=%.0f", cacheDuration.Seconds())

	handler := func(w http.ResponseWriter) {
		expiresValue := time.Now().UTC().Add(expiresDuration).Format(http.TimeFormat)

		w.Header().Set("Expires", expiresValue)
		w.Header().Set("Cache-Control", maxAgeValue)
	}

	testRequestsCachedDuration(t, handler, cacheDuration)
}

// Should cache responses with a status code of 404. It's a common
// misconception that 404 responses shouldn't be cached; they should because
// they can be expensive to generate.
func TestCache404Response(t *testing.T) {
	ResetBackends(backendsByPriority)

	handler := func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusNotFound)
	}

	testRequestsCachedIndefinite(t, handler)
}

// Should cache multiple distinct responses for the same URL when origin responds
// with a `Vary` header and clients provide requests with different values
// for that header.
func TestCacheVary(t *testing.T) {
	ResetBackends(backendsByPriority)

	if vendorCloudflare {
		t.Skip(notSupportedByVendor)
	}

	const reqHeaderName = "CustomThing"
	const respHeaderName = "Reflected-" + reqHeaderName
	headerVals := []string{
		"first distinct",
		"second distinct",
		"third distinct",
	}

	req := NewUniqueEdgeGET(t)

	for _, populateCache := range []bool{true, false} {
		for _, headerVal := range headerVals {
			if populateCache {
				originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Vary", reqHeaderName)
					w.Header().Set(respHeaderName, r.Header.Get(reqHeaderName))
				})
			} else {
				originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
					t.Error("Request should not have made it to origin")
					w.Header().Set(respHeaderName, "not cached")
				})
			}

			req.Header.Set(reqHeaderName, headerVal)
			resp := RoundTripCheckError(t, req)
			defer resp.Body.Close()

			if recVal := resp.Header.Get(respHeaderName); recVal != headerVal {
				t.Errorf(
					"Request received wrong %q header. Expected %q, got %q",
					respHeaderName,
					headerVal,
					recVal,
				)
			}
		}
	}
}

// Should deliver gzip compressed responses bodies to client requests with
// the header `Accept-Encoding: gzip` and plaintext response bodies for
// clients that don't. Some vendors:
//   - appear to implment this independent of normal `Vary` headers
//   - will make a single request w/gzip to origin and handle
//     compression/decompression to the client themselves.
func TestCacheAcceptEncodingGzip(t *testing.T) {
	ResetBackends(backendsByPriority)

	const expectedBody = "may or may not be gzipped"
	var reqAcceptEncoding string
	var expectedContentEncoding string

	// Tell the transport not to add Accept-Encoding headers and automatically
	// decompress responses. Restore the setting after the test.
	origClientDisableCompression := client.DisableCompression
	client.DisableCompression = true
	defer func() {
		client.DisableCompression = origClientDisableCompression
	}()

	req := NewUniqueEdgeGET(t)

	for _, populateCache := range []bool{true, false} {
		for _, gzipContent := range []bool{false, true} {
			if gzipContent {
				reqAcceptEncoding = "gzip"
				expectedContentEncoding = "gzip"
			} else {
				reqAcceptEncoding = "somethingelse"
				expectedContentEncoding = ""
			}

			if populateCache {
				originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
					// NB: Some vendors don't appear to depend on this.
					w.Header().Set("Vary", "Accept-Encoding")

					// Don't switch on `gzipContent` because the edge may ask for gzip
					// even if the client hasn't.
					if r.Header.Get("Accept-Encoding") == "gzip" {
						gzbuf := new(bytes.Buffer)
						gzwriter := gzip.NewWriter(gzbuf)
						gzwriter.Write([]byte(expectedBody))
						gzwriter.Close()

						w.Header().Set("Content-Encoding", "gzip")
						w.Header().Set("Content-Type", "text/plain; charset=utf-8")

						w.Write(gzbuf.Bytes())
					} else {
						w.Write([]byte(expectedBody))
					}
				})
			} else {
				originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
					t.Error("Request should not have made it to origin")
					w.Write([]byte("uncached response"))
				})
			}

			req.Header.Set("Accept-Encoding", reqAcceptEncoding)
			resp := RoundTripCheckError(t, req)
			defer resp.Body.Close()

			if headerVal := resp.Header.Get("Content-Encoding"); headerVal != expectedContentEncoding {
				t.Fatalf(
					"Request received incorrect Content-Encoding header. Expected %q, got %q",
					expectedContentEncoding,
					headerVal,
				)
			}

			var rawBody io.ReadCloser
			if gzipContent {
				var err error
				rawBody, err = gzip.NewReader(resp.Body)
				if err != nil {
					t.Fatal(err)
				}
				defer rawBody.Close()
			} else {
				rawBody = resp.Body
			}

			body, err := ioutil.ReadAll(rawBody)
			if err != nil {
				t.Fatal(err)
			}

			if bodyStr := string(body); bodyStr != expectedBody {
				t.Errorf(
					"Request received incorrect response body. Expected %q, got %q",
					expectedBody,
					bodyStr,
				)
			}
		}
	}
}

// Should cache distinct responses for requests with the same path but
// different query params.
func TestCacheUniqueQueryParams(t *testing.T) {
	ResetBackends(backendsByPriority)

	const respHeaderName = "Request-RawQuery"

	req1 := NewUniqueEdgeGET(t)
	req2 := NewUniqueEdgeGET(t)

	if req1.URL.Path != req2.URL.Path {
		t.Fatalf(
			"Request paths do not match. Expected %q, got %q",
			req1.URL.Path,
			req2.URL.Path,
		)
	}
	if req1.URL.RawQuery == req2.URL.RawQuery {
		t.Fatalf(
			"Request query params do not differ. Expected %q != %q",
			req1.URL.RawQuery,
			req2.URL.RawQuery,
		)
	}

	for _, populateCache := range []bool{true, false} {
		for _, req := range []*http.Request{req1, req2} {
			if populateCache {
				originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set(respHeaderName, r.URL.RawQuery)
				})
			} else {
				originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
					t.Errorf(
						"Request with query param %q should not have made it to origin",
						r.URL.RawQuery,
					)
				})
			}

			resp := RoundTripCheckError(t, req)
			defer resp.Body.Close()

			if recVal := resp.Header.Get(respHeaderName); recVal != req.URL.RawQuery {
				t.Errorf(
					"Request received wrong %q header. Expected %q, got %q",
					respHeaderName,
					req.URL.RawQuery,
					recVal,
				)
			}
		}
	}
}

// Should cache distinct responses for requests with the same query params
// but paths of different case-sensitivity.
func TestCacheUniqueCaseSensitive(t *testing.T) {
	ResetBackends(backendsByPriority)

	const reqPath = "/CaseSensitive"
	const respHeaderName = "Request-Path"

	req1 := NewUniqueEdgeGET(t)
	req2 := NewUniqueEdgeGET(t)

	req1.URL.Path = strings.ToLower(reqPath)
	req2.URL.Path = strings.ToUpper(reqPath)
	req1.URL.RawQuery = req2.URL.RawQuery

	if req1.URL.Path == req2.URL.Path {
		t.Fatalf(
			"Request paths do not differ. Expected %q != %q",
			req1.URL.Path,
			req2.URL.Path,
		)
	}
	if req1.URL.RawQuery != req2.URL.RawQuery {
		t.Fatalf(
			"Request query params do not match. Expected %q, got %q",
			req1.URL.RawQuery,
			req2.URL.RawQuery,
		)
	}

	for _, populateCache := range []bool{true, false} {
		for _, req := range []*http.Request{req1, req2} {
			if populateCache {
				originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set(respHeaderName, r.URL.Path)
				})
			} else {
				originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
					t.Errorf(
						"Request with path %q should not have made it to origin",
						r.URL.Path,
					)
				})
			}

			resp := RoundTripCheckError(t, req)
			defer resp.Body.Close()

			if recVal := resp.Header.Get(respHeaderName); recVal != req.URL.Path {
				t.Errorf(
					"Request received wrong %q header. Expected %q, got %q",
					respHeaderName,
					req.URL.Path,
					recVal,
				)
			}
		}
	}
}
