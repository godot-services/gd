package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
	return dir
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
}

// newFakeLibrary serves asset 1234 ("Dialogic 2") whose download is /zip.
func newFakeLibrary(t *testing.T) *fakeLibrary {
	t.Helper()
	lib := &fakeLibrary{zip: zipBytes(t, testZip), zipStatus: http.StatusOK}
	lib.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/asset/1234":
			fmt.Fprintf(w, `{"title":"Dialogic 2","version_string":"2.0","download_url":"%s/zip"}`, lib.URL)
		case "/zip":
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

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
