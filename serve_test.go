package serve

import (
	"archive/zip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		"post": {
			method: "POST",
			target: "/asdf",
			status: http.StatusOK,
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

func TestArchive(t *testing.T) {
	type testCase struct {
		files     map[string]string
		filenames []string
	}
	for name, tc := range map[string]testCase{
		"flat files": {
			files: map[string]string{
				"asdf": "this is the content of asdf",
				"qwer": "this is the content of qwer",
			},
			filenames: []string{"asdf"},
		},
		"whole dir": {
			files: map[string]string{
				"asdf": "this is the content of asdf",
				"qwer": "this is the content of qwer",
			},
			filenames: []string{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatalf("Cannot create temp directory: %v", err)
			}

			for name, content := range tc.files {
				f, err := os.Create(filepath.Join(tmpDir, name))
				if err != nil {
					t.Fatalf("Cannot create temp file: %v", err)
				}
				defer f.Close()

				_, err = f.Write([]byte(content))
				if err != nil {
					t.Fatalf("Cannot write to temp file: %v", err)
				}
			}

			zipPath := filepath.Join(os.TempDir(), "test.zip")
			f, err := os.Create(zipPath)
			if err != nil {
				t.Fatalf("Cannot create test.zip file: %v", err)
			}
			defer func() {
				f.Close()

				if err := os.Remove(zipPath); err != nil {
					t.Fatalf("Cannot remove test.zip file: %v", err)
				}
			}()

			if err := archive(f, tmpDir, tc.filenames); err != nil {
				t.Errorf("Error should be nil; got %v", err)
			}

			// check buffer's content
			r, err := zip.OpenReader(zipPath)
			if err != nil {
				t.Fatalf("Cannot open test.zip file: %v", err)
			}
			defer r.Close()

			filenames := tc.filenames
			if len(tc.filenames) == 0 {
				for name := range tc.files {
					filenames = append(filenames, name)
				}
			}

			if len(r.File) != len(filenames) {
				t.Errorf("Number of files in archive should be %v; got %v", len(filenames), len(r.File))
			}

			contains := func(arr []string, val string) bool {
				for _, s := range arr {
					if s == val {
						return true
					}
				}
				return false
			}

			for _, f := range r.File {
				if !contains(filenames, f.Name) {
					t.Errorf("%v should not be in archive", f.Name)
				}

				rc, err := f.Open()
				if err != nil {
					t.Fatalf("Cannot open %v in archive: %v", f.Name, err)
				}
				defer rc.Close()

				b, err := ioutil.ReadAll(rc)
				if err != nil {
					t.Fatalf("Cannot read %v in archive: %v", f.Name, err)
				}

				if string(b) != tc.files[f.Name] {
					t.Errorf("%v should contain `%v`; got `%v`", f.Name, tc.files[f.Name], string(b))
				}
			}
		})
	}
}
