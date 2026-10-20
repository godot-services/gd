package cmd

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/godot-services/gd/internal/godot"
)

func TestRemoveAsset(t *testing.T) {
	dir := newTestProject(t)
	lib := newFakeLibrary(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}

	if err := run(t, lib, "remove", "dialogic_2"); err != nil {
		t.Fatal(err)
	}

	pkg, _ := godot.LoadPackage(dir)
	lock, _ := godot.LoadLock(dir)
	if exists(filepath.Join(dir, "addons", "x")) || len(pkg.AllAssets()) != 0 || len(lock) != 0 {
		t.Fatalf("pkg %+v lock %+v", pkg.AllAssets(), lock)
	}
}

func TestRemoveUnknown(t *testing.T) {
	newTestProject(t)
	if err := run(t, newFakeLibrary(t), "remove", "nope"); !errors.Is(err, godot.ErrAssetNotFound) {
		t.Fatal(err)
	}
}
