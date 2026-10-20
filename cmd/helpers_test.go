package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// newTestProject creates a Godot project in a temp dir and changes into it.
func newTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "project.godot"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)
	isolateCache(t)
	return dir
}

// isolateCache points os.UserCacheDir at a temp dir (HOME on macOS, XDG_CACHE_HOME on Linux).
func isolateCache(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	return home
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write([]byte(content))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

var testZip = map[string]string{"dialogic-main/addons/x/plugin.gd": "extends Node", "dialogic-main/README.md": "readme"}

type fakeLibrary struct {
	*httptest.Server
	zip       []byte
	zipStatus int
	zipHits   int

	// asset 1234
	version     string
	staticURL   bool // download_url stays /zip when version changes
	assetStatus int
	assetHits   int

	// GET /asset search
	results   []map[string]string
	lastQuery url.Values
}

// newFakeLibrary serves asset 1234 ("Dialogic 2") whose download is /zip?v=<version>.
func newFakeLibrary(t *testing.T) *fakeLibrary {
	t.Helper()
	lib := &fakeLibrary{zip: zipBytes(t, testZip), zipStatus: http.StatusOK, version: "2.0", assetStatus: http.StatusOK}
	lib.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/asset/1234":
			lib.assetHits++
			w.WriteHeader(lib.assetStatus)
			download := lib.URL + "/zip?v=" + lib.version
			if lib.staticURL {
				download = lib.URL + "/zip"
			}
			fmt.Fprintf(w, `{"title":"Dialogic 2","version_string":"%s","download_url":"%s"}`, lib.version, download)
		case "/asset":
			lib.lastQuery = r.URL.Query()
			json.NewEncoder(w).Encode(map[string]any{"result": lib.results, "total_items": len(lib.results)})
		case "/zip":
			lib.zipHits++
			w.WriteHeader(lib.zipStatus)
			w.Write(lib.zip)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(lib.Close)
	return lib
}

func run(t *testing.T, lib *fakeLibrary, args ...string) error {
	t.Helper()
	root := newRootCmd()
	root.SetArgs(append([]string{"--library-url", lib.URL}, args...))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	return root.Execute()
}

// runOut is run with the command output captured.
func runOut(t *testing.T, lib *fakeLibrary, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	root := newRootCmd()
	root.SetArgs(append([]string{"--library-url", lib.URL}, args...))
	root.SetOut(&out)
	root.SetErr(io.Discard)
	err := root.Execute()
	return out.String(), err
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
