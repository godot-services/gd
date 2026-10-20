package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheInfoAndClean(t *testing.T) {
	newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	dir, err := cacheDir()
	if err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(dir, "notes.txt")
	os.WriteFile(keep, []byte("not a zip"), 0o644)

	out, err := runOut(t, lib, "cache", "info")
	if err != nil || !strings.Contains(out, dir) || !strings.Contains(out, "1 file") {
		t.Fatal(out, err)
	}

	out, err = runOut(t, lib, "cache", "clean")
	if err != nil || !strings.Contains(out, "removed 1 file") {
		t.Fatal(out, err)
	}
	zips, _ := filepath.Glob(filepath.Join(dir, "*.zip"))
	if len(zips) != 0 || !exists(keep) {
		t.Fatalf("zips left %v, other file kept %v", zips, exists(keep))
	}
}

func TestCacheInfoWithoutCacheDir(t *testing.T) {
	isolateCache(t)
	out, err := runOut(t, newFakeLibrary(t), "cache", "info")
	if err != nil || !strings.Contains(out, "0 files") {
		t.Fatal(out, err)
	}
}
