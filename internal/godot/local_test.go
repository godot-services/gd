package godot

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseTag(t *testing.T) {
	for in, want := range map[string]string{
		"4.7.2":        "4.7.2-stable",
		"4.7":          "4.7-stable",
		"4.8-beta1":    "4.8-beta1",
		"4.7-rc2":      "4.7-rc2",
		"4.8-dev7":     "4.8-dev7",
		"4.7.2-stable": "4.7.2-stable",
	} {
		if got := ReleaseTag(in); got != want {
			t.Errorf("ReleaseTag(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMajorMinor(t *testing.T) {
	for in, want := range map[string]string{"4.7.2": "4.7", "4.7.2-stable": "4.7", "4.8-beta1": "4.8", "4.3": "4.3"} {
		if got := MajorMinor(in); got != want {
			t.Errorf("MajorMinor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewPackageRejectsInvalidGodotVersion(t *testing.T) {
	for _, v := range []string{"3.6", "4.x", "4.7.2-alpha"} {
		_, err := NewPackage([]byte("[editor]\nversion=" + v + "\n"))
		if !errors.Is(err, ErrInvalidGodotVersion) {
			t.Errorf("version %q: err = %v, want ErrInvalidGodotVersion", v, err)
		}
	}
	if _, err := NewPackage([]byte("[editor]\nversion=4.8-beta1\n")); err != nil {
		t.Errorf("valid version rejected: %v", err)
	}
}

func TestEditorDotnetParsed(t *testing.T) {
	p, err := NewPackage([]byte("[editor]\nversion=4.2.1\ndotnet=true\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !p.Editor.Dotnet || p.Editor.Version != "4.2.1" {
		t.Errorf("editor = %+v", p.Editor)
	}
}

func TestLocalEditorRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if got, err := LoadLocalEditor(dir); err != nil || got != "" {
		t.Fatalf("missing file: %q, %v", got, err)
	}
	if err := SaveLocalEditor(dir, "4.7.2-dotnet"); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadLocalEditor(dir); err != nil || got != "4.7.2-dotnet" {
		t.Fatalf("load: %q, %v", got, err)
	}
	if err := SaveLocalEditor(dir, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, LocalFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file not removed: %v", err)
	}
	if err := SaveLocalEditor(dir, ""); err != nil {
		t.Errorf("removing missing file: %v", err)
	}
}

func TestMainScene(t *testing.T) {
	for _, tc := range []struct{ content, want string }{
		{"[application]\nrun/main_scene=\"res://main.tscn\"\n", "res://main.tscn"},
		{"[application]\nrun/main_scene=\"uid://abc\"\n", "uid://abc"},
	} {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ProjectFileName), []byte(tc.content), 0o644)
		got, err := (&Project{Location: dir}).MainScene()
		if err != nil || got != tc.want {
			t.Errorf("MainScene = %q, %v; want %q", got, err, tc.want)
		}
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ProjectFileName), []byte("[application]\nconfig/name=\"x\"\n"), 0o644)
	if _, err := (&Project{Location: dir}).MainScene(); !errors.Is(err, ErrNoMainScene) {
		t.Errorf("err = %v, want ErrNoMainScene", err)
	}
}

func TestLoadLocalEditorSpecFormat(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, LocalFileName), []byte("[editor]\nname=\"custombuild-x\"\n"), 0o644)
	if got, err := LoadLocalEditor(dir); err != nil || got != "custombuild-x" {
		t.Fatalf("got %q, %v", got, err)
	}
}
