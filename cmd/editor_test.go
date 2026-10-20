package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godot-services/gd/internal/editor"
)

func runEditor(t *testing.T, builds string, args ...string) (string, error) {
	t.Helper()
	return runOut(t, newFakeLibrary(t), append([]string{"--builds-url", builds}, args...)...)
}

func loadReg(t *testing.T) (*editor.Registry, string) {
	t.Helper()
	dir, err := editor.DataDir()
	if err != nil {
		t.Fatal(err)
	}
	r, err := editor.LoadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	return r, dir
}

func TestEditorInstallFromPackage(t *testing.T) {
	dir := newTestProject(t)
	os.WriteFile(filepath.Join(dir, "package.godot"), []byte("[editor]\nversion=4.7.2\n"), 0o644)
	out, err := runEditor(t, newFakeBuilds(t), "editor", "install")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := loadReg(t)
	e := r.Editors["4.7.2"]
	if e.Version != "4.7.2-stable" || e.Dotnet || r.Default != "4.7.2" || !exists(e.Path) {
		t.Errorf("registry = %+v default %q", e, r.Default)
	}
	if !strings.Contains(out, "installed 4.7.2 (4.7.2-stable) -> "+e.Path) {
		t.Errorf("out = %q", out)
	}
}

func TestEditorInstallNoVersion(t *testing.T) {
	newTestProject(t)
	if _, err := runEditor(t, newFakeBuilds(t), "editor", "install"); err == nil {
		t.Fatal("expected error")
	}
}

func TestEditorInstallDotnetKeepsDefault(t *testing.T) {
	newTestProject(t)
	b := newFakeBuilds(t)
	for _, a := range [][]string{{"4.7.2"}, {"4.7.2", "--dotnet"}} {
		if _, err := runEditor(t, b, append([]string{"editor", "install"}, a...)...); err != nil {
			t.Fatal(err)
		}
	}
	r, _ := loadReg(t)
	if !r.Editors["4.7.2-dotnet"].Dotnet || r.Default != "4.7.2" {
		t.Errorf("registry = %+v", r)
	}
}

func TestEditorInstallAlready(t *testing.T) {
	newTestProject(t)
	b := newFakeBuilds(t)
	runEditor(t, b, "editor", "install", "4.7.2")
	out, err := runEditor(t, b, "editor", "install", "4.7.2")
	if err != nil || !strings.Contains(out, "already installed") {
		t.Errorf("out %q err %v", out, err)
	}
}

func TestEditorUninstall(t *testing.T) {
	newTestProject(t)
	b := newFakeBuilds(t)
	runEditor(t, b, "editor", "install", "4.7.2")
	if _, err := runEditor(t, b, "editor", "uninstall", "4.7.2"); err != nil {
		t.Fatal(err)
	}
	r, dir := loadReg(t)
	if len(r.Editors) != 0 || r.Default != "" || exists(filepath.Join(dir, "editors", "4.7.2")) {
		t.Errorf("not removed: %+v", r)
	}
	if _, err := runEditor(t, b, "editor", "uninstall", "4.7.2"); err == nil {
		t.Error("unknown should fail")
	}
}

func TestEditorUninstallLinked(t *testing.T) {
	newTestProject(t)
	r, dir := loadReg(t)
	keep := filepath.Join(t.TempDir(), "godot")
	os.WriteFile(keep, nil, 0o755)
	r.Editors["mine"] = editor.Editor{Name: "mine", Path: keep, Source: "linked", Build: "custom"}
	r.Save(dir)
	_, err := runEditor(t, "", "editor", "uninstall", "mine")
	if err == nil || !strings.Contains(err.Error(), "gd editor unlink mine") || !exists(keep) {
		t.Errorf("err %v", err)
	}
}

func TestEditorList(t *testing.T) {
	dir := newTestProject(t)
	out, _ := runEditor(t, "", "editor", "list")
	if !strings.Contains(out, "no editors installed, run gd editor install <version>") {
		t.Errorf("out = %q", out)
	}
	b := newFakeBuilds(t)
	runEditor(t, b, "editor", "install", "4.7.2")
	runEditor(t, b, "editor", "install", "4.8", "--dotnet")
	os.WriteFile(filepath.Join(dir, "package.godot"), []byte("[editor]\nversion=4.8\ndotnet=true\n"), 0o644)
	out, err := runEditor(t, b, "editor", "list")
	if err != nil {
		t.Fatal(err)
	}
	var def, proj string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "4.7.2-stable") {
			def = l
		}
		if strings.Contains(l, "4.8-dotnet") {
			proj = l
		}
	}
	if !strings.HasPrefix(def, "*") || !strings.HasPrefix(proj, ">") || !strings.Contains(out, "NAME") {
		t.Errorf("out = %q", out)
	}
}
