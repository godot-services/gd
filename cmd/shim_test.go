package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/godot-services/gd/internal/editor"
	"github.com/godot-services/gd/internal/testutil/fakegodot"
)

func TestShimFile(t *testing.T) {
	name, content := shimFile("linux", "/opt/it's/gd")
	if name != "godot" || !strings.HasPrefix(content, "#!/bin/sh\n") ||
		!strings.Contains(content, `if [ -x '/opt/it'\''s/gd' ]; then exec '/opt/it'\''s/gd' godot -- "$@"; fi`) ||
		!strings.HasSuffix(content, "exec gd godot -- \"$@\"\n") {
		t.Errorf("unix %s:\n%s", name, content)
	}

	name, content = shimFile("windows", `C:\Program Files (x86)\100%\gd.exe`)
	if name != "godot.cmd" || !strings.Contains(content, `if not exist "C:\Program Files (x86)\100%%\gd.exe" goto path`+"\r\n") ||
		!strings.Contains(content, `"C:\Program Files (x86)\100%%\gd.exe" godot -- %*`+"\r\n") ||
		!strings.Contains(content, ":path\r\ngd godot -- %*\r\n") {
		t.Errorf("windows %s:\n%s", name, content)
	}
}

func shimPath(t *testing.T) string {
	t.Helper()
	dir, err := editor.DataDir()
	if err != nil {
		t.Fatal(err)
	}
	name, _ := shimFile(runtime.GOOS, "")
	return filepath.Join(dir, "bin", name)
}

func TestShimInstallAndUninstall(t *testing.T) {
	isolateCache(t)
	t.Setenv("PATH", "")
	out, err := runEditor(t, "", "editor", "shim", "install")
	path := shimPath(t)
	exe, _ := os.Executable()
	if err != nil || !strings.Contains(out, "installed "+path) || !strings.Contains(out, "add "+filepath.Dir(path)+" to your PATH") {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	b, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(b), exe) {
		t.Fatalf("shim = %q, err = %v", b, err)
	}
	if fi, _ := os.Stat(path); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v", fi.Mode())
	}

	t.Setenv("PATH", filepath.Dir(path))
	if out, err := runEditor(t, "", "editor", "shim", "install"); err != nil || strings.Contains(out, "to your PATH") {
		t.Errorf("on PATH: out = %q, err = %v", out, err)
	}

	if out, err := runEditor(t, "", "editor", "shim", "uninstall"); err != nil || !strings.Contains(out, "removed "+path) || exists(path) {
		t.Errorf("uninstall: out = %q, err = %v", out, err)
	}
	if out, err := runEditor(t, "", "editor", "shim", "uninstall"); err != nil || !strings.Contains(out, "no shim installed") {
		t.Errorf("second uninstall: out = %q, err = %v", out, err)
	}
}

// The fake godot stands in for gd here: it logs the arguments the shim passes on.
func TestShimRunsGd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the unix shim is run on unix")
	}
	_, log := fakegodot.Setup(t, "")
	exe, _ := os.Executable()
	dir := t.TempDir()
	write := func(target string) string {
		_, content := shimFile(runtime.GOOS, target)
		path := filepath.Join(dir, "godot")
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
		return path
	}

	if out, err := exec.Command(write(exe), "a b", "c").CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if got := readLog(t, log); got != "godot\t--\ta b\tc\n" {
		t.Errorf("absolute: log = %q", got)
	}

	// gd moved away: the shim falls back to gd on PATH
	bin := t.TempDir()
	if err := os.Symlink(exe, filepath.Join(bin, "gd")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	os.Remove(log)
	if out, err := exec.Command(write(filepath.Join(dir, "missing", "gd")), "x").CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if got := readLog(t, log); got != "godot\t--\tx\n" {
		t.Errorf("fallback: log = %q", got)
	}
}

func TestShimPrefersGdOnPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	isolateCache(t)
	exe, _ := os.Executable()
	bin := t.TempDir()
	link := filepath.Join(bin, "gd")
	if err := os.Symlink(exe, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if out, err := runEditor(t, "", "editor", "shim", "install"); err != nil || !strings.Contains(out, "-> "+link) {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	if b, _ := os.ReadFile(shimPath(t)); !strings.Contains(string(b), "'"+link+"'") {
		t.Errorf("shim = %s", b)
	}
}

// Linking the shim as an editor would make every godot run start gd again, forever.
func TestEditorLinkRejectsShim(t *testing.T) {
	isolateCache(t)
	if _, err := runEditor(t, "", "editor", "shim", "install"); err != nil {
		t.Fatal(err)
	}
	_, err := runEditor(t, "", "editor", "link", "sys", shimPath(t))
	if err == nil || !strings.Contains(err.Error(), "shim") {
		t.Fatalf("err = %v", err)
	}
	if r, _ := loadReg(t); len(r.Editors) != 0 {
		t.Errorf("registered %+v", r.Editors)
	}
}
