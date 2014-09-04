// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fake_httptest_test

import (
	"fmt"
	"io/ioutil"
	"log"

	"net/http/httptest"
)

func ExampleResponseRecorder() {
	handler := func(w fake_http.ResponseWriter, r *fake_http.Request) {
		fake_http.Error(w, "something failed", fake_http.StatusInternalServerError)
	}

	req, err := fake_http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	handler(w, req)

	fmt.Printf("%d - %s", w.Code, w.Body.String())
	// Output: 500 - something failed
}

func ExampleServer() {
	ts := httptest.NewServer(fake_http.HandlerFunc(func(w fake_http.ResponseWriter, r *fake_http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	res, err := fake_http.Get(ts.URL)
	if err != nil {
		log.Fatal(err)
	}
	greeting, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s", greeting)
	// Output: Hello, client
}
