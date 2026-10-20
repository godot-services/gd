package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/install"
)

func TestInstallFromLock(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	before, _ := godot.LoadLock(dir)
	os.RemoveAll(filepath.Join(dir, "addons"))

	if err := run(t, lib, "install"); err != nil {
		t.Fatal(err)
	}

	after, _ := godot.LoadLock(dir)
	if !exists(filepath.Join(dir, "addons", "x", "plugin.gd")) || !reflect.DeepEqual(before, after) {
		t.Fatalf("before %+v after %+v", before, after)
	}
}

func TestInstallTwiceIsIdempotent(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := run(t, lib, "install"); err != nil {
			t.Fatal(i, err)
		}
	}
	if !exists(filepath.Join(dir, "addons", "x", "plugin.gd")) {
		t.Fatal("addon missing")
	}
}

func TestInstallHashMismatch(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	lib.zip = zipBytes(t, map[string]string{"dialogic-main/addons/x/plugin.gd": "tampered"})
	isolateCache(t) // empty cache, so the tampered zip is downloaded

	if err := run(t, lib, "install"); !errors.Is(err, install.ErrHashMismatch) {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "addons", "x", "plugin.gd"))
	if string(b) != "extends Node" {
		t.Fatalf("addon changed: %s", b)
	}
}

func TestInstallResolvesMissingLockEntry(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	os.WriteFile(filepath.Join(dir, godot.PackageFileName), []byte("[assets]\ndialogic={ library=\"1234\" }\n"), 0o644)

	if err := run(t, lib, "install"); err != nil {
		t.Fatal(err)
	}
	lock, _ := godot.LoadLock(dir)
	if lock["dialogic"].SHA256 == "" || !exists(filepath.Join(dir, "addons", "x")) {
		t.Fatalf("%+v", lock)
	}
}

func TestInstallRemovesStaleLockEntry(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	os.MkdirAll(filepath.Join(dir, "addons", "old"), 0o755)
	godot.Lock{"old": {Source: "library:1", URL: "u", SHA256: "s", Paths: []string{"addons/old"}}}.Save(dir)

	if err := run(t, lib, "install"); err != nil {
		t.Fatal(err)
	}
	lock, _ := godot.LoadLock(dir)
	if exists(filepath.Join(dir, "addons", "old")) || len(lock) != 0 {
		t.Fatalf("%+v", lock)
	}
}

func TestInstallRelocksChangedSource(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	os.WriteFile(filepath.Join(dir, godot.PackageFileName), []byte("[assets]\ndialogic={ library=\"1234\" }\n"), 0o644)
	godot.Lock{"dialogic": {Source: "library:1", URL: lib.URL + "/old", SHA256: "s", Paths: []string{"addons/x"}}}.Save(dir)

	if err := run(t, lib, "install"); err != nil {
		t.Fatal(err)
	}
	lock, _ := godot.LoadLock(dir)
	if lock["dialogic"].Source != "library:1234" {
		t.Fatalf("%+v", lock)
	}
}

func TestInstallAfterRename(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, godot.PackageFileName), []byte("[assets]\ndlg={ library=\"1234\" }\n"), 0o644)

	if err := run(t, lib, "install"); err != nil {
		t.Fatal(err)
	}
	lock, _ := godot.LoadLock(dir)
	if _, ok := lock["dialogic_2"]; ok || !reflect.DeepEqual(lock["dlg"].Paths, []string{"addons/x"}) || !exists(filepath.Join(dir, "addons", "x")) {
		t.Fatalf("%+v", lock)
	}
}

func TestInstallRemovesPathsNoLongerShipped(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	os.MkdirAll(filepath.Join(dir, "addons", "old"), 0o755)
	os.WriteFile(filepath.Join(dir, godot.PackageFileName), []byte("[assets]\ndialogic={ library=\"1234\" }\n"), 0o644)
	godot.Lock{"dialogic": {Source: "library:1", URL: "u", SHA256: "s", Paths: []string{"addons/old"}}}.Save(dir)

	if err := run(t, lib, "install"); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(dir, "addons", "old")) || !exists(filepath.Join(dir, "addons", "x")) {
		t.Fatal("old path not cleaned up")
	}
}

func TestInstallRefusesLockEntryWithoutHash(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	lock, _ := godot.LoadLock(dir)
	entry := lock["dialogic_2"]
	entry.SHA256 = ""
	lock["dialogic_2"] = entry
	lock.Save(dir)
	os.RemoveAll(filepath.Join(dir, "addons"))

	if err := run(t, lib, "install"); !errors.Is(err, errLockWithoutHash) {
		t.Fatal(err)
	}
	if exists(filepath.Join(dir, "addons", "x")) {
		t.Fatal("installed without verification")
	}
}

func TestInstallUsesCache(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(filepath.Join(dir, "addons"))

	if err := run(t, lib, "install"); err != nil {
		t.Fatal(err)
	}
	if lib.zipHits != 1 || !exists(filepath.Join(dir, "addons", "x", "plugin.gd")) {
		t.Fatalf("zip downloaded %d times", lib.zipHits)
	}
}

func TestInstallNoDevSkipsDevAssets(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	os.WriteFile(filepath.Join(dir, godot.PackageFileName), []byte("[assets]\nrun={ library=\"1234\" }\n\n[dev_assets]\ntool={ library=\"1234\" }\n"), 0o644)
	devEntry := godot.LockEntry{Source: "library:1234", URL: "u", SHA256: "s", Paths: []string{"addons/devtool"}}
	godot.Lock{"tool": devEntry}.Save(dir)
	os.MkdirAll(filepath.Join(dir, "addons", "devtool"), 0o755)

	if err := run(t, lib, "install", "--no-dev"); err != nil {
		t.Fatal(err)
	}

	lock, _ := godot.LoadLock(dir)
	if !exists(filepath.Join(dir, "addons", "x")) || !exists(filepath.Join(dir, "addons", "devtool")) {
		t.Fatal("wrong folders")
	}
	if !reflect.DeepEqual(lock["tool"], devEntry) || lock["run"].SHA256 == "" || lib.assetHits != 1 {
		t.Fatalf("lock %+v, asset requests %d", lock, lib.assetHits)
	}
}
