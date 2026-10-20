package cmd

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godot-services/gd/internal/godot"
)

func TestOutdatedReportsNewVersion(t *testing.T) {
	newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	lib.version = "2.1"

	out, err := runOut(t, lib, "outdated")
	if !errors.Is(err, errOutdated) || !strings.Contains(out, "dialogic_2") || !strings.Contains(out, "2.0") || !strings.Contains(out, "2.1") {
		t.Fatal(out, err)
	}
}

func TestOutdatedUpToDate(t *testing.T) {
	newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}

	out, err := runOut(t, lib, "outdated")
	if err != nil || !strings.Contains(out, "all assets are up to date") {
		t.Fatal(out, err)
	}
}

func TestOutdatedShowsPinnedAndNotInstalled(t *testing.T) {
	dir := newTestProject(t)
	os.WriteFile(filepath.Join(dir, godot.PackageFileName), []byte("[assets]\ntool={ git=\"https://github.com/a/tool\", version=\"v1\" }\ndlg={ library=\"1234\" }\n"), 0o644)
	godot.Lock{"tool": {Source: "git:https://github.com/a/tool@v1", URL: "u", SHA256: "s", Paths: []string{"addons/tool"}}}.Save(dir)

	out, err := runOut(t, newFakeLibrary(t), "outdated")
	if err != nil || !strings.Contains(out, "pinned") || !strings.Contains(out, "not installed") {
		t.Fatal(out, err)
	}
}

func TestOutdatedLibraryError(t *testing.T) {
	newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	lib.assetStatus = http.StatusNotFound

	if _, err := runOut(t, lib, "outdated"); err == nil || errors.Is(err, errOutdated) {
		t.Fatal(err)
	}
}

func TestOutdatedWritesNothing(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	before := readBoth(t, dir)
	lib.version = "2.1"

	runOut(t, lib, "outdated")
	if readBoth(t, dir) != before {
		t.Fatal("files changed")
	}
}

func readBoth(t *testing.T, dir string) string {
	t.Helper()
	a, _ := os.ReadFile(filepath.Join(dir, godot.PackageFileName))
	b, _ := os.ReadFile(filepath.Join(dir, godot.LockFileName))
	return string(a) + string(b)
}

func TestOutdatedSourceChangedNeedsInstall(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	os.WriteFile(filepath.Join(dir, godot.PackageFileName), []byte("[assets]\ndlg={ library=\"1234\" }\n"), 0o644)
	godot.Lock{"dlg": {Source: "git:https://github.com/a/dlg@v1", URL: "u", SHA256: "s", VersionString: "v1", Paths: []string{"addons/x"}}}.Save(dir)

	out, err := runOut(t, lib, "outdated")
	if err != nil || !strings.Contains(out, "run gd install") || lib.assetHits != 0 {
		t.Fatal(out, err, lib.assetHits)
	}
}
