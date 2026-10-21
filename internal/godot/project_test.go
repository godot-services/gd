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

func TestGodotVersion(t *testing.T) {
	cases := []struct {
		name, src, want string
		err             error
	}{
		{"godot 4", "[application]\nconfig/name=\"G\"\nconfig/features=PackedStringArray(\"4.3\", \"Forward Plus\")\n", "4.3", nil},
		{"patch", "[application]\nconfig/features=PackedStringArray(\"4.2.1\")\n", "4.2.1", nil},
		{"version not first", "[application]\nconfig/features=PackedStringArray(\"Forward Plus\", \"4.3\")\n", "4.3", nil},
		{"godot 3", "[application]\nconfig/name=\"G\"\n", "", ErrNoGodotVersion},
		{"no version entry", "[application]\nconfig/features=PackedStringArray(\"Mobile\")\n", "", ErrNoGodotVersion},
		{"features after multi-line string", "[application]\nconfig/description=\"x\nconfig/features=PackedStringArray(\\\"9.9\\\")\"\nconfig/features=PackedStringArray(\"4.5\")\n", "4.5", nil},
		{"multi-line values around", "[input]\nui_x={\n\"deadzone\": 0.5,\n\"events\": []\n}\n[application]\nconfig/features=PackedStringArray(\"4.4\")\n", "4.4", nil},
	}
	for _, c := range cases {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ProjectFileName), []byte(c.src), 0o644)
		p, err := NewProject(dir)
		if err != nil {
			t.Fatal(err)
		}
		got, err := p.GodotVersion()
		if got != c.want || !errors.Is(err, c.err) {
			t.Errorf("%s: got %q, %v; want %q, %v", c.name, got, err, c.want, c.err)
		}
	}
}

func TestMainSceneFromFull(t *testing.T) {
	dir := t.TempDir()
	src, err := os.ReadFile("testdata/projects/full.godot")
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, ProjectFileName), src, 0o644)
	p, err := NewProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.MainScene()
	if err != nil || got != "res://main.tscn" {
		t.Errorf("got %q, %v", got, err)
	}
}
