// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fake_httptest

import (
	"io/ioutil"
	"testing"

	"../fake_http/"
)

func TestServer(t *testing.T) {
	ts := NewServer(fake_http.HandlerFunc(func(w fake_http.ResponseWriter, r *fake_http.Request) {
		w.Write([]byte("hello"))
	}))
	defer ts.Close()
	res, err := fake_http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q, want hello", string(got))
	}
}
