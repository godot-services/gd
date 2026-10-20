package cmd

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/godot-services/gd/internal/godot"
)

func TestAddLibraryAsset(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)

	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}

	if !exists(filepath.Join(dir, "addons", "x", "plugin.gd")) || exists(filepath.Join(dir, "README.md")) {
		t.Fatal("addon not installed correctly")
	}
	pkg, err := godot.LoadPackage(dir)
	if err != nil || pkg.Assets["dialogic_2"].Library != "1234" {
		t.Fatal(pkg, err)
	}
	lock, err := godot.LoadLock(dir)
	entry := lock["dialogic_2"]
	if err != nil || entry.SHA256 == "" || entry.Source != "library:1234" || !reflect.DeepEqual(entry.Paths, []string{"addons/x"}) {
		t.Fatalf("%+v %v", entry, err)
	}
}

func TestAddDevAssetWithName(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)

	if err := run(t, lib, "add", "1234", "--name", "dlg", "--dev"); err != nil {
		t.Fatal(err)
	}
	pkg, _ := godot.LoadPackage(dir)
	if pkg.DevAssets["dlg"].Library != "1234" {
		t.Fatal(pkg.DevAssets)
	}
}

func TestAddGitAssetRequiresVersion(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)

	err := run(t, lib, "add", "https://github.com/a/b")
	if !errors.Is(err, godot.ErrGitVersionRequired) || !strings.Contains(err.Error(), "--version") {
		t.Fatal(err)
	}
	if exists(filepath.Join(dir, godot.PackageFileName)) || exists(filepath.Join(dir, "addons")) {
		t.Fatal("files written")
	}
}

func TestAddDuplicateName(t *testing.T) {
	newTestProject(t)
	lib := newFakeLibrary(t)

	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	if err := run(t, lib, "add", "1234"); !errors.Is(err, godot.ErrAssetExists) {
		t.Fatal(err)
	}
}

func TestAddOutsideProject(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	lib := newFakeLibrary(t)

	if err := run(t, lib, "add", "1234"); !errors.Is(err, godot.ErrNoGodotProject) {
		t.Fatal(err)
	}
	if exists(filepath.Join(dir, godot.PackageFileName)) {
		t.Fatal("package.godot created")
	}
}

func TestAddFailingDownloadLeavesProjectUntouched(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	lib.zipStatus = http.StatusInternalServerError

	if err := run(t, lib, "add", "1234"); err == nil {
		t.Fatal("expected error")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("project changed: %v", entries)
	}
}

func TestAddInvalidReference(t *testing.T) {
	newTestProject(t)
	if err := run(t, newFakeLibrary(t), "add", "dialogic"); err == nil {
		t.Fatal("expected error")
	}
}

func TestAddFailingPackageSaveIsRecoverable(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	pkgFile := filepath.Join(dir, godot.PackageFileName)
	os.WriteFile(pkgFile, []byte("[assets]\n"), 0o444)

	if err := run(t, lib, "add", "1234"); err == nil {
		t.Fatal("expected save error")
	}
	os.Chmod(pkgFile, 0o644)

	// install cleans up the half state, so adding again works
	if err := run(t, lib, "install"); err != nil {
		t.Fatal(err)
	}
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
}

func TestAddRejectsInvalidName(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)

	for _, name := range []string{"DEFAULT", "a b", "a.b"} {
		if err := run(t, lib, "add", "1234", "--name", name); !errors.Is(err, errInvalidName) {
			t.Errorf("%q: %v", name, err)
		}
	}
	if exists(filepath.Join(dir, "addons")) {
		t.Fatal("installed despite invalid name")
	}
}

func TestAddRejectsVersionForLibraryAsset(t *testing.T) {
	dir := newTestProject(t)
	err := run(t, newFakeLibrary(t), "add", "1234", "--version", "1.0")
	if !errors.Is(err, errVersionForLibrary) {
		t.Fatal(err)
	}
	if exists(filepath.Join(dir, "addons")) {
		t.Fatal("installed")
	}
}
