package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/testutil/fakegodot"
)

// setupFakeEditor links the fake godot as the default editor and returns its invocation log.
func setupFakeEditor(t *testing.T) string {
	t.Helper()
	_, log := fakegodot.Setup(t, "4.7.2.stable.official.abc")
	if _, err := linkEditor(t, "fake"); err != nil {
		t.Fatal(err)
	}
	if _, err := runEditor(t, "", "editor", "default", "fake"); err != nil {
		t.Fatal(err)
	}
	return log
}

func readLog(t *testing.T, log string) string {
	t.Helper()
	b, _ := os.ReadFile(log)
	return string(b)
}

func TestOpenRunsEditorWithProject(t *testing.T) {
	dir, _ := filepath.EvalSymlinks(newTestProject(t))
	log := setupFakeEditor(t)
	if _, err := runEditor(t, "", "editor", "open"); err != nil {
		t.Fatal(err)
	}
	if got := readLog(t, log); !strings.Contains(got, "-e\t--path\t"+dir) {
		t.Errorf("log = %q", got)
	}
}

func TestOpenShortcut(t *testing.T) {
	dir, _ := filepath.EvalSymlinks(newTestProject(t))
	log := setupFakeEditor(t)
	if _, err := runEditor(t, "", "open"); err != nil {
		t.Fatal(err)
	}
	if got := readLog(t, log); !strings.Contains(got, "-e\t--path\t"+dir) {
		t.Errorf("log = %q", got)
	}
}

func TestOpenDetachReturnsImmediately(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("detach is tested on unix")
	}
	newTestProject(t)
	setupFakeEditor(t)
	t.Setenv("GD_FAKE_GODOT_SLEEP", "2s")
	start := time.Now()
	if _, err := runEditor(t, "", "open", "--detach"); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("took %v", d)
	}
}

func TestGodotPassesArgs(t *testing.T) {
	newTestProject(t)
	log := setupFakeEditor(t)
	if _, err := runEditor(t, "", "godot", "--", "--headless", "--version"); err != nil {
		t.Fatal(err)
	}
	if got := readLog(t, log); !strings.Contains(got, "--headless\t--version\n") {
		t.Errorf("log = %q", got)
	}
}

func TestGodotPassesExitCode(t *testing.T) {
	newTestProject(t)
	setupFakeEditor(t)
	t.Setenv("GD_FAKE_GODOT_EXIT", "3")
	_, err := runEditor(t, "", "godot", "--", "x")
	var ee exitError
	if !errors.As(err, &ee) || ee.code != 3 {
		t.Errorf("err = %v", err)
	}
}

func TestOpenWithoutEditorFails(t *testing.T) {
	newTestProject(t)
	_, err := runEditor(t, "", "open")
	if err == nil || !strings.Contains(err.Error(), "gd editor default") {
		t.Errorf("err = %v", err)
	}
}

func TestOpenWarnsOnLocalMismatch(t *testing.T) {
	dir := newTestProject(t)
	setupFakeEditor(t)
	os.WriteFile(dir+"/package.godot", []byte("[editor]\nversion=4.8\n"), 0o644)
	os.WriteFile(dir+"/package.local.godot", []byte("[editor]\nname=fake\n"), 0o644)
	var errOut bytes.Buffer
	c := newRootCmd()
	c.SetErr(&errOut)
	_, e, err := resolveProjectEditor(c)
	if err != nil || e.Name != "fake" || !strings.Contains(errOut.String(), "differs from [editor]") {
		t.Errorf("e=%v err=%v stderr=%q", e, err, errOut.String())
	}
}

func TestExitCode(t *testing.T) {
	var buf bytes.Buffer
	if c := exitCode(exitError{3}, &buf); c != 3 || buf.Len() != 0 {
		t.Errorf("exitError: code %d out %q", c, buf.String())
	}
	if c := exitCode(errors.New("boom"), &buf); c != 1 || buf.String() != "Error: boom\n" {
		t.Errorf("other: code %d out %q", c, buf.String())
	}
	if c := exitCode(nil, &buf); c != 0 {
		t.Errorf("nil: code %d", c)
	}
}

func TestGodotOutsideProjectUsesDefault(t *testing.T) {
	isolateCache(t)
	log := setupFakeEditor(t)
	chdir(t, t.TempDir())
	if _, err := runEditor(t, "", "godot", "--", "--headless", "x"); err != nil {
		t.Fatal(err)
	}
	if got := readLog(t, log); !strings.Contains(got, "\n--headless\tx\n") {
		t.Errorf("log = %q", got)
	}
	t.Setenv("GD_FAKE_GODOT_EXIT", "4")
	var ee exitError
	if _, err := runEditor(t, "", "godot", "--", "x"); !errors.As(err, &ee) || ee.code != 4 {
		t.Errorf("err = %v", err)
	}
	if _, err := runEditor(t, "", "start"); !errors.Is(err, godot.ErrNoGodotProject) {
		t.Errorf("start err = %v", err)
	}
}

func TestOpenOutsideProjectOpensProjectManager(t *testing.T) {
	isolateCache(t)
	log := setupFakeEditor(t)
	chdir(t, t.TempDir())
	if _, err := runEditor(t, "", "open"); err != nil {
		t.Fatal(err)
	}
	if got := readLog(t, log); !strings.HasSuffix(got, "\n--project-manager\n") {
		t.Errorf("log = %q", got)
	}
}

func TestOpenOutsideProjectWithoutDefaultFails(t *testing.T) {
	isolateCache(t)
	chdir(t, t.TempDir())
	if _, err := runEditor(t, "", "open"); err == nil || !strings.Contains(err.Error(), "gd editor default") {
		t.Errorf("err = %v", err)
	}
}
