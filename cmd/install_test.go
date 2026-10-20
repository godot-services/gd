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
