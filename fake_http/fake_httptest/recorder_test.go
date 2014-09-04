// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fake_httptest

import (
	"fmt"
	"testing"

	"../fake_http/"
)

func TestRecorder(t *testing.T) {
	type checkFunc func(*ResponseRecorder) error
	check := func(fns ...checkFunc) []checkFunc { return fns }

	hasStatus := func(wantCode int) checkFunc {
		return func(rec *ResponseRecorder) error {
			if rec.Code != wantCode {
				return fmt.Errorf("Status = %d; want %d", rec.Code, wantCode)
			}
			return nil
		}
	}
	hasContents := func(want string) checkFunc {
		return func(rec *ResponseRecorder) error {
			if rec.Body.String() != want {
				return fmt.Errorf("wrote = %q; want %q", rec.Body.String(), want)
			}
			return nil
		}
	}
	hasFlush := func(want bool) checkFunc {
		return func(rec *ResponseRecorder) error {
			if rec.Flushed != want {
				return fmt.Errorf("Flushed = %v; want %v", rec.Flushed, want)
			}
			return nil
		}
	}

	tests := []struct {
		name   string
		h      func(w fake_http.ResponseWriter, r *fake_http.Request)
		checks []checkFunc
	}{
		{
			"200 default",
			func(w fake_http.ResponseWriter, r *fake_http.Request) {},
			check(hasStatus(200), hasContents("")),
		},
		{
			"first code only",
			func(w fake_http.ResponseWriter, r *fake_http.Request) {
				w.WriteHeader(201)
				w.WriteHeader(202)
				w.Write([]byte("hi"))
			},
			check(hasStatus(201), hasContents("hi")),
		},
		{
			"write sends 200",
			func(w fake_http.ResponseWriter, r *fake_http.Request) {
				w.Write([]byte("hi first"))
				w.WriteHeader(201)
				w.WriteHeader(202)
			},
			check(hasStatus(200), hasContents("hi first"), hasFlush(false)),
		},
		{
			"flush",
			func(w fake_http.ResponseWriter, r *fake_http.Request) {
				w.(fake_http.Flusher).Flush() // also sends a 200
				w.WriteHeader(201)
			},
			check(hasStatus(200), hasFlush(true)),
		},
	}
	r, _ := fake_http.NewRequest("GET", "http://foo.com/", nil)
	for _, tt := range tests {
		h := fake_http.HandlerFunc(tt.h)
		rec := NewRecorder()
		h.ServeHTTP(rec, r)
		for _, check := range tt.checks {
			if err := check(rec); err != nil {
				t.Errorf("%s: %v", tt.name, err)
			}
		}
	}
}
