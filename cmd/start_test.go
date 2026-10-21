package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func writeMainScene(t *testing.T, dir string) {
	t.Helper()
	os.WriteFile(filepath.Join(dir, "project.godot"), []byte("[application]\nrun/main_scene=\"res://main.tscn\"\n"), 0o644)
}

func TestStartRunsMainScene(t *testing.T) {
	dir, _ := filepath.EvalSymlinks(newTestProject(t))
	writeMainScene(t, dir)
	log := setupFakeEditor(t)
	if _, err := runEditor(t, "", "start"); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimPrefix(readLog(t, log), "--version\n"); got != "--path\t"+dir+"\n" {
		t.Errorf("log = %q", got)
	}
}

func TestStartWithoutMainSceneFails(t *testing.T) {
	newTestProject(t)
	setupFakeEditor(t)
	_, err := runEditor(t, "", "start")
	if err == nil || !strings.Contains(err.Error(), "run/main_scene") {
		t.Errorf("err = %v", err)
	}
}

func TestStartRelativeScene(t *testing.T) {
	dir, _ := filepath.EvalSymlinks(newTestProject(t))
	os.MkdirAll(filepath.Join(dir, "levels"), 0o755)
	os.WriteFile(filepath.Join(dir, "levels", "one.tscn"), nil, 0o644)
	log := setupFakeEditor(t)
	if _, err := runEditor(t, "", "start", filepath.Join("levels", "one.tscn")); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimPrefix(readLog(t, log), "--version\n"); got != "--path\t"+dir+"\tres://levels/one.tscn\n" {
		t.Errorf("log = %q", got)
	}
}

func TestStartScenePathErrors(t *testing.T) {
	newTestProject(t)
	setupFakeEditor(t)
	for _, s := range []string{"nope.tscn", "../x.tscn"} {
		if _, err := runEditor(t, "", "start", s); err == nil || !strings.Contains(err.Error(), s) {
			t.Errorf("%s: err = %v", s, err)
		}
	}
}

func TestStartPassesThroughURIs(t *testing.T) {
	dir, _ := filepath.EvalSymlinks(newTestProject(t))
	log := setupFakeEditor(t)
	for _, s := range []string{"uid://abc123", "res://a.tscn"} {
		if _, err := runEditor(t, "", "start", s); err != nil {
			t.Fatal(err)
		}
	}
	want := "--path\t" + dir + "\tuid://abc123\n--path\t" + dir + "\tres://a.tscn\n"
	if got := strings.TrimPrefix(readLog(t, log), "--version\n"); got != want {
		t.Errorf("log = %q", got)
	}
}

func TestStartDetach(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("detach is tested on unix")
	}
	dir := newTestProject(t)
	writeMainScene(t, dir)
	setupFakeEditor(t)
	t.Setenv("GD_FAKE_GODOT_SLEEP", "2s")
	start := time.Now()
	if _, err := runEditor(t, "", "start", "--detach"); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("took %v", d)
	}
}
