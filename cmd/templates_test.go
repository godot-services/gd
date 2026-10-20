package cmd

import (
	"errors"
	"github.com/godot-services/gd/internal/godot"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godot-services/gd/internal/editor"
)

func testTemplatesDir(t *testing.T) string {
	t.Helper()
	d, err := editor.GodotDataDir()
	if err != nil {
		t.Fatal(err)
	}
	return editor.TemplatesDir(d)
}

func TestTemplatesInstallListUninstall(t *testing.T) {
	dir := newTestProject(t)
	os.WriteFile(filepath.Join(dir, "package.godot"), []byte("[editor]\nversion=4.7.2\n"), 0o644)
	b := newFakeBuilds(t)
	out, err := runEditor(t, b, "templates", "install")
	if err != nil || !strings.Contains(out, "installed templates 4.7.2.stable -> ") {
		t.Fatalf("out %q err %v", out, err)
	}
	if out, err = runEditor(t, b, "templates", "install", "--dotnet"); err != nil || !strings.Contains(out, "4.7.2.stable.mono") {
		t.Fatalf("out %q err %v", out, err)
	}
	if !exists(filepath.Join(testTemplatesDir(t), "4.7.2.stable.mono", "linux_release.x86_64")) {
		t.Error("mono templates missing")
	}
	out, _ = runEditor(t, b, "templates", "list")
	if out != "4.7.2.stable\n4.7.2.stable.mono\n" {
		t.Errorf("list = %q", out)
	}
	if _, err := runEditor(t, b, "templates", "uninstall", "../x"); err == nil {
		t.Error("want error for ../x")
	}
	if _, err := runEditor(t, b, "templates", "uninstall", "4.7.2.stable"); err != nil {
		t.Fatal(err)
	}
	if out, _ = runEditor(t, b, "templates", "list"); out != "4.7.2.stable.mono\n" {
		t.Errorf("list = %q", out)
	}
}

func TestTemplatesListEmpty(t *testing.T) {
	newTestProject(t)
	out, err := runEditor(t, newFakeBuilds(t), "templates", "list")
	if err != nil || strings.TrimSpace(out) != "no export templates installed" {
		t.Errorf("out %q err %v", out, err)
	}
}

func TestTemplatesInstallSkipsExisting(t *testing.T) {
	newTestProject(t)
	b := newFakeBuilds(t)
	runEditor(t, b, "templates", "install", "4.7.2")
	marker := filepath.Join(testTemplatesDir(t), "4.7.2.stable", "marker")
	os.WriteFile(marker, nil, 0o644)
	out, err := runEditor(t, b, "templates", "install", "4.7.2")
	if err != nil || !strings.Contains(out, "templates 4.7.2.stable already installed (use --force)") || !exists(marker) {
		t.Errorf("out %q err %v", out, err)
	}
	if _, err := runEditor(t, b, "templates", "install", "4.7.2", "--force"); err != nil || exists(marker) {
		t.Errorf("force: err %v, marker still there %v", err, exists(marker))
	}
}

func TestTemplatesInstallNeedsVersion(t *testing.T) {
	newTestProject(t)
	if _, err := runEditor(t, newFakeBuilds(t), "templates", "install"); err == nil {
		t.Error("want error without version")
	}
}

func TestTemplatesInstallSelfContainedEditor(t *testing.T) {
	dir := newTestProject(t)
	os.WriteFile(filepath.Join(dir, "package.godot"), []byte("[editor]\nversion=4.7.2\n"), 0o644)
	sc := t.TempDir()
	os.WriteFile(filepath.Join(sc, "_sc_"), nil, 0o644)
	dd, _ := editor.DataDir()
	reg := &editor.Registry{Default: "mine", Editors: map[string]editor.Editor{
		"mine": {Name: "mine", Path: filepath.Join(sc, "godot"), Version: "4.7.2-stable", Build: "official", Source: "linked"},
	}}
	if err := reg.Save(dd); err != nil {
		t.Fatal(err)
	}
	if _, err := runEditor(t, newFakeBuilds(t), "templates", "install"); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(sc, "editor_data", "export_templates", "4.7.2.stable", "version.txt")) {
		t.Error("not installed next to the self-contained editor")
	}
	if exists(filepath.Join(testTemplatesDir(t), "4.7.2.stable")) {
		t.Error("also installed in the Godot data dir")
	}
}

func TestInstallCommandsRejectInvalidVersion(t *testing.T) {
	newTestProject(t)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer srv.Close()
	for _, c := range [][]string{{"templates", "install"}, {"editor", "install"}} {
		for _, v := range []string{"3.5.3", "4.7.2-foo1", "../x", "3.6"} {
			_, err := runEditor(t, srv.URL, append(c, v)...)
			if !errors.Is(err, godot.ErrInvalidGodotVersion) {
				t.Errorf("%v %s: err = %v", c, v, err)
			}
		}
	}
	if hits != 0 || exists(filepath.Join(filepath.Dir(testTemplatesDir(t)), "x")) {
		t.Errorf("hits = %d", hits)
	}
}
