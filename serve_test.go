package serve

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFromArgs(t *testing.T) {
	type testCase struct {
		args  []string
		err   error
		dir   string
		addr  string
		quiet bool
	}
	for name, tc := range map[string]testCase{
		"default values": {
			args:  []string{},
			err:   nil,
			dir:   ".",
			addr:  "localhost:9876",
			quiet: false,
		},
		"dir": {
			args:  []string{"/home"},
			err:   nil,
			dir:   "/home",
			addr:  "localhost:9876",
			quiet: false,
		},
		"flags": {
			args:  []string{"-a", "127.0.0.1:1234", "-q"},
			err:   nil,
			dir:   ".",
			addr:  "127.0.0.1:1234",
			quiet: true,
		},
		"dir and flags": {
			args:  []string{"-a", "127.0.0.1:1234", "-q", "/home"},
			err:   nil,
			dir:   "/home",
			addr:  "127.0.0.1:1234",
			quiet: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var app app
			err := app.fromArgs(tc.args)
			if err != tc.err {
				t.Errorf("err should be %v; got %v", tc.err, err)
			}
			if app.dir != tc.dir {
				t.Errorf("dir should be %v; got %v", tc.dir, app.dir)
			}
			if app.addr != tc.addr {
				t.Errorf("addr should be %v; got %v", tc.addr, app.addr)
			}
			if app.quiet != tc.quiet {
				t.Errorf("quiet should be %v; got %v", tc.quiet, app.quiet)
			}
		})
	}
}

func TestHandler(t *testing.T) {
	type testCase struct {
		method string
		target string
		body   string

		status int
	}
	for name, tc := range map[string]testCase{
		"get": {
			method: "GET",
			target: "/",
			status: http.StatusOK,
		},
		"get not existing": {
			method: "GET",
			target: "/asdf",
			status: http.StatusNotFound,
		},
		"put": {
			method: "PUT",
			target: "/",
			status: http.StatusMethodNotAllowed,
		},
		"patch": {
			method: "PATCH",
			target: "/",
			status: http.StatusMethodNotAllowed,
		},
		"delete": {
			method: "DELETE",
			target: "/",
			status: http.StatusMethodNotAllowed,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var app app
			h := app.handler()

			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			resp := httptest.NewRecorder()

			h(resp, req)

			if resp.Code != tc.status {
				t.Errorf("StatusCode should be %v; got %v", tc.status, resp.Code)
			}
		})
	}
}
