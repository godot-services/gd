package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/godot-services/gd/internal/godot"
)

func TestEndToEnd(t *testing.T) {
	lib := newFakeLibrary(t)

	a := newTestProject(t)
	if err := run(t, lib, "add", "1234"); err != nil {
		t.Fatal(err)
	}

	// fresh checkout: only the two files are committed
	b := newTestProject(t)
	for _, f := range []string{godot.PackageFileName, godot.LockFileName} {
		data, err := os.ReadFile(filepath.Join(a, f))
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(b, f), data, 0o644)
	}

	if err := run(t, lib, "install"); err != nil {
		t.Fatal(err)
	}
	want, _ := os.ReadFile(filepath.Join(a, "addons", "x", "plugin.gd"))
	got, err := os.ReadFile(filepath.Join(b, "addons", "x", "plugin.gd"))
	if err != nil || string(got) != string(want) {
		t.Fatalf("got %q want %q (%v)", got, want, err)
	}

	if err := run(t, lib, "remove", "dialogic_2"); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(b, "addons", "x")) || exists(filepath.Join(b, godot.LockFileName)) {
		t.Fatal("not cleaned up")
	}
}
