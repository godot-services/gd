package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/install"
)

func TestUpdateAll(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	before, _ := godot.LoadLock(dir)
	pkgBefore, _ := os.ReadFile(filepath.Join(dir, godot.PackageFileName))
	lib.version = "2.1"
	lib.zip = zipBytes(t, map[string]string{"dialogic-main/addons/x/plugin.gd": "v2.1"})

	out, err := runOut(t, lib, "update")
	if err != nil || !strings.Contains(out, "updated dialogic_2 2.0 -> 2.1") {
		t.Fatal(out, err)
	}
	after, _ := godot.LoadLock(dir)
	content, _ := os.ReadFile(filepath.Join(dir, "addons", "x", "plugin.gd"))
	pkgAfter, _ := os.ReadFile(filepath.Join(dir, godot.PackageFileName))
	if after["dialogic_2"].VersionString != "2.1" || after["dialogic_2"].SHA256 == before["dialogic_2"].SHA256 || string(content) != "v2.1" || string(pkgAfter) != string(pkgBefore) {
		t.Fatalf("%+v %q", after, content)
	}
}

func TestUpdateNamedOnly(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234", "--name", "a"); err != nil {
		t.Fatal(err)
	}
	f, _ := os.OpenFile(filepath.Join(dir, godot.PackageFileName), os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString("\n[assets.b]\ngit=\"https://github.com/a/b\"\nversion=\"v1\"\n")
	f.Close()
	lock, _ := godot.LoadLock(dir)
	gitEntry := godot.LockEntry{Source: "git:https://github.com/a/b@v1", URL: "u", SHA256: "s", Paths: []string{"addons/b"}}
	lock["b"] = gitEntry
	lock.Save(dir)
	hits := lib.assetHits

	if err := run(t, lib, "update", "a"); err != nil {
		t.Fatal(err)
	}
	after, _ := godot.LoadLock(dir)
	if lib.assetHits != hits+1 || !reflect.DeepEqual(after["b"], gitEntry) {
		t.Fatalf("hits %d, %+v", lib.assetHits-hits, after["b"])
	}
}

func TestUpdateUnknownName(t *testing.T) {
	newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "update", "nope"); !errors.Is(err, godot.ErrAssetNotFound) || lib.assetHits != 0 {
		t.Fatal(err, lib.assetHits)
	}
}

func TestUpdateUpToDate(t *testing.T) {
	newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	out, err := runOut(t, lib, "update")
	if err != nil || !strings.Contains(out, "dialogic_2 is up to date") {
		t.Fatal(out, err)
	}
}

func TestUpdateReplacesChangedFolders(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	lib.version = "3.0"
	lib.zip = zipBytes(t, map[string]string{"dialogic-main/addons/y/plugin.gd": "y"})

	if err := run(t, lib, "update"); err != nil {
		t.Fatal(err)
	}
	lock, _ := godot.LoadLock(dir)
	if exists(filepath.Join(dir, "addons", "x")) || !exists(filepath.Join(dir, "addons", "y")) || !reflect.DeepEqual(lock["dialogic_2"].Paths, []string{"addons/y"}) {
		t.Fatalf("%+v", lock)
	}
}

func TestUpdateReportsNewVersionStringWithSameURL(t *testing.T) {
	newTestProject(t)
	lib := newFakeLibrary(t)
	lib.staticURL = true
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	lib.version = "2.1"

	out, err := runOut(t, lib, "update")
	if err != nil || !strings.Contains(out, "updated dialogic_2 2.0 -> 2.1") {
		t.Fatal(out, err)
	}
}

func TestUpdateWithoutLockEntryShowsDash(t *testing.T) {
	dir := newTestProject(t)
	os.WriteFile(filepath.Join(dir, godot.PackageFileName), []byte("[assets]\ndialogic_2={ library=\"1234\" }\n"), 0o644)

	out, err := runOut(t, newFakeLibrary(t), "update")
	if err != nil || !strings.Contains(out, "updated dialogic_2 - -> 2.0") {
		t.Fatal(out, err)
	}
}

func TestUpdateAfterRenameHintsInstall(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, godot.PackageFileName), []byte("[assets]\ndlg={ library=\"1234\" }\n"), 0o644)

	err := run(t, lib, "update")
	if !errors.Is(err, install.ErrForeignPath) || !strings.Contains(err.Error(), "gd install") {
		t.Fatal(err)
	}
}
