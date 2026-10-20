package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConsolePath(t *testing.T) {
	dir := t.TempDir()
	gui := filepath.Join(dir, "Godot_v4.7.2-stable_win64.exe")
	if got := ConsolePath(gui); got != gui {
		t.Errorf("no console file: %q", got)
	}
	console := filepath.Join(dir, "Godot_v4.7.2-stable_win64_console.exe")
	os.WriteFile(console, nil, 0o755)
	if got := ConsolePath(gui); got != console {
		t.Errorf("got %q", got)
	}
}
