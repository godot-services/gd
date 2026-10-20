package godot

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newProjectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ProjectFileName), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestNewProjectInDirWithProjectFile(t *testing.T) {
	dir := newProjectDir(t)
	p, err := NewProject(dir)
	if err != nil || p.Location != dir {
		t.Fatal(p, err)
	}
}

func TestNewProjectFromFileLocation(t *testing.T) {
	dir := newProjectDir(t)
	p, err := NewProject(filepath.Join(dir, ProjectFileName))
	if err != nil || p.Location != dir {
		t.Fatal(p, err)
	}
}

func TestNewProjectWithoutProjectFile(t *testing.T) {
	_, err := NewProject(t.TempDir())
	if !errors.Is(err, ErrNoGodotProject) {
		t.Fatal(err)
	}
}

func TestNewProjectUnknownLocation(t *testing.T) {
	_, err := NewProject(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}
