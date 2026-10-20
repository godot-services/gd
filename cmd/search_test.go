package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godot-services/gd/internal/godot"
)

var gutResult = map[string]string{"asset_id": "1709", "title": "GUT", "version_string": "9.6.1", "godot_version": "4.3", "cost": "MIT", "support_level": "community"}

func newGodot4Project(t *testing.T, version string) string {
	t.Helper()
	dir := newTestProject(t)
	os.WriteFile(filepath.Join(dir, "project.godot"), []byte("[application]\nconfig/features=PackedStringArray(\""+version+"\", \"Forward Plus\")\n"), 0o644)
	return dir
}

func TestSearchUsesProjectGodotVersion(t *testing.T) {
	newGodot4Project(t, "4.3")
	lib := newFakeLibrary(t)
	lib.results = []map[string]string{gutResult}

	out, err := runOut(t, lib, "search", "gut")
	if err != nil || lib.lastQuery.Get("godot_version") != "4.3" || lib.lastQuery.Get("filter") != "gut" || lib.lastQuery.Get("max_results") != "20" {
		t.Fatal(lib.lastQuery, err)
	}
	if !strings.Contains(out, "GUT") || !strings.Contains(out, "1709") || !strings.Contains(out, "1 of 1 results") {
		t.Fatal(out)
	}
}

func TestSearchGodotFlagOverrides(t *testing.T) {
	newGodot4Project(t, "4.3")
	lib := newFakeLibrary(t)

	if _, err := runOut(t, lib, "search", "gut", "--godot", "4.2"); err != nil || lib.lastQuery.Get("godot_version") != "4.2" {
		t.Fatal(lib.lastQuery, err)
	}
}

func TestSearchOutsideProjectWithGodotFlag(t *testing.T) {
	chdir(t, t.TempDir())
	if _, err := runOut(t, newFakeLibrary(t), "search", "gut", "--godot", "4.3"); err != nil {
		t.Fatal(err)
	}
}

func TestSearchOutsideProjectWithoutGodotFlag(t *testing.T) {
	chdir(t, t.TempDir())
	if _, err := runOut(t, newFakeLibrary(t), "search", "gut"); !errors.Is(err, godot.ErrNoGodotProject) {
		t.Fatal(err)
	}
}

func TestSearchInvalidMax(t *testing.T) {
	newGodot4Project(t, "4.3")
	lib := newFakeLibrary(t)
	for _, max := range []string{"0", "501"} {
		if _, err := runOut(t, lib, "search", "gut", "--max", max); err == nil {
			t.Errorf("--max %s: expected error", max)
		}
	}
	if lib.lastQuery != nil {
		t.Fatal("request sent")
	}
}

func TestSearchNoResults(t *testing.T) {
	newGodot4Project(t, "4.3")
	out, err := runOut(t, newFakeLibrary(t), "search", "nothing")
	if err != nil || !strings.Contains(out, `no assets found for "nothing" (Godot 4.3)`) {
		t.Fatal(out, err)
	}
}
