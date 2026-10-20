package editor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/godot-services/gd/internal/godot"
)

func tpz(t *testing.T, version string) []byte {
	return makeZip(t, zent{"templates/version.txt", version + "\n", 0o644}, zent{"templates/linux_release.x86_64", "tpl", 0o644})
}

func TestInstallTemplates(t *testing.T) {
	for _, tt := range []struct {
		dotnet         bool
		asset, version string
	}{
		{false, "Godot_v4.7.2-stable_export_templates.tpz", "4.7.2.stable"},
		{true, "Godot_v4.7.2-stable_mono_export_templates.tpz", "4.7.2.stable.mono"},
	} {
		c := server(t, tt.asset, tpz(t, tt.version), "")
		dir := filepath.Join(t.TempDir(), "export_templates")
		got, err := InstallTemplates(context.Background(), c, "4.7.2-stable", tt.dotnet, dir, false)
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(dir, tt.version); got != want {
			t.Errorf("path = %s, want %s", got, want)
		}
		if b, err := os.ReadFile(filepath.Join(got, "linux_release.x86_64")); err != nil || string(b) != "tpl" {
			t.Errorf("template file: %q %v", b, err)
		}
		if m, _ := filepath.Glob(filepath.Join(dir, ".staging-*")); len(m) != 0 {
			t.Errorf("staging left: %v", m)
		}
		if _, err := InstallTemplates(context.Background(), c, "4.7.2-stable", tt.dotnet, dir, false); !errors.Is(err, ErrTemplatesInstalled) {
			t.Errorf("second install err = %v", err)
		}
	}
}

func TestInstallTemplatesSkipsExistingWithoutDownload(t *testing.T) {
	// the client points nowhere: any request would fail with a different error
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "4.7.2.stable"), 0o755)
	_, err := InstallTemplates(context.Background(), server(t, "x", nil, ""), "4.7.2-stable", false, dir, false)
	if !errors.Is(err, ErrTemplatesInstalled) {
		t.Fatalf("err = %v", err)
	}
}

func TestInstallTemplatesForce(t *testing.T) {
	c := server(t, "Godot_v4.7.2-stable_export_templates.tpz", tpz(t, "4.7.2.stable"), "")
	dir := t.TempDir()
	old := filepath.Join(dir, "4.7.2.stable")
	os.MkdirAll(old, 0o755)
	os.WriteFile(filepath.Join(old, "stale"), nil, 0o644)
	if _, err := InstallTemplates(context.Background(), c, "4.7.2-stable", false, dir, true); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(old, "stale")) || !exists(filepath.Join(old, "version.txt")) {
		t.Error("force did not replace the old dir")
	}
}

func TestInstallTemplatesBadInput(t *testing.T) {
	asset := "Godot_v4.7.2-stable_export_templates.tpz"
	dir := t.TempDir()
	good := tpz(t, "4.7.2.stable")
	if _, err := InstallTemplates(context.Background(), server(t, asset, good, "00"), "4.7.2-stable", false, dir, false); err == nil {
		t.Error("want hash mismatch")
	}
	for _, v := range []string{"../evil", "a/b"} {
		if _, err := InstallTemplates(context.Background(), server(t, asset, tpz(t, v), ""), "4.7.2-stable", false, dir, false); err == nil {
			t.Errorf("version.txt %q accepted", v)
		}
	}
	if _, err := InstallTemplates(context.Background(), server(t, asset, makeZip(t, zent{"templates/x", "x", 0o644}), ""), "4.7.2-stable", false, dir, false); err == nil {
		t.Error("missing version.txt accepted")
	}
	if m, _ := filepath.Glob(filepath.Join(dir, "*")); len(m) != 0 {
		t.Errorf("leftovers: %v", m)
	}
	if m, _ := filepath.Glob(filepath.Join(dir, ".staging-*")); len(m) != 0 {
		t.Errorf("staging left: %v", m)
	}
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func TestListUninstallTemplates(t *testing.T) {
	dir := t.TempDir()
	if got, err := ListTemplates(filepath.Join(dir, "none")); err != nil || len(got) != 0 {
		t.Errorf("missing dir: %v %v", got, err)
	}
	for _, n := range []string{"4.7.2.stable", "4.7.2.stable.mono"} {
		os.MkdirAll(filepath.Join(dir, n), 0o755)
	}
	os.MkdirAll(filepath.Join(dir, ".staging-x"), 0o755)
	if got, _ := ListTemplates(dir); !slices.Equal(got, []string{"4.7.2.stable", "4.7.2.stable.mono"}) {
		t.Errorf("list = %v", got)
	}
	for _, bad := range []string{"../x", "a/b", "..", ".", ""} {
		if err := UninstallTemplates(dir, bad); err == nil {
			t.Errorf("uninstall %q accepted", bad)
		}
	}
	if err := UninstallTemplates(dir, "nope"); err == nil {
		t.Error("unknown accepted")
	}
	if err := UninstallTemplates(dir, "4.7.2.stable"); err != nil || exists(filepath.Join(dir, "4.7.2.stable")) {
		t.Errorf("uninstall: %v", err)
	}
}

func TestSelfContainedTemplatesDir(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "godot")
	if _, ok := SelfContainedTemplatesDir(bin); ok {
		t.Error("no marker, want false")
	}
	os.WriteFile(filepath.Join(root, "_sc_"), nil, 0o644)
	if d, ok := SelfContainedTemplatesDir(bin); !ok || d != filepath.Join(root, "editor_data", "export_templates") {
		t.Errorf("got %s %v", d, ok)
	}
	app := t.TempDir()
	macBin := filepath.Join(app, "Godot.app", "Contents", "MacOS", "Godot")
	os.WriteFile(filepath.Join(app, "._sc_"), nil, 0o644)
	if d, ok := SelfContainedTemplatesDir(macBin); !ok || d != filepath.Join(app, "editor_data", "export_templates") {
		t.Errorf("mac got %s %v", d, ok)
	}
}

func TestTemplatesDir(t *testing.T) {
	if got := TemplatesDir("/g"); got != filepath.Join("/g", "export_templates") {
		t.Error(got)
	}
}

func TestInstallTemplatesVersionMismatch(t *testing.T) {
	c := server(t, "Godot_v4.7.2-stable_export_templates.tpz", tpz(t, "4.6.0.stable"), "")
	dir := t.TempDir()
	old := filepath.Join(dir, "4.6.0.stable")
	os.MkdirAll(old, 0o755)
	os.WriteFile(filepath.Join(old, "keep"), nil, 0o644)
	if _, err := InstallTemplates(context.Background(), c, "4.7.2-stable", false, dir, true); err == nil {
		t.Fatal("want mismatch error")
	}
	if !exists(filepath.Join(old, "keep")) || exists(filepath.Join(dir, "4.7.2.stable")) {
		t.Error("mismatch touched the templates dir")
	}
}

func TestInstallTemplatesForceKeepsOnlyNew(t *testing.T) {
	c := server(t, "Godot_v4.7.2-stable_export_templates.tpz", tpz(t, "4.7.2.stable"), "")
	dir := t.TempDir()
	old := filepath.Join(dir, "4.7.2.stable")
	os.MkdirAll(old, 0o755)
	os.WriteFile(filepath.Join(old, "stale"), nil, 0o644)
	if _, err := InstallTemplates(context.Background(), c, "4.7.2-stable", false, dir, true); err != nil {
		t.Fatal(err)
	}
	m, _ := filepath.Glob(filepath.Join(dir, "*"))
	h, _ := filepath.Glob(filepath.Join(dir, ".*"))
	if len(m) != 1 || len(h) != 0 || exists(filepath.Join(old, "stale")) {
		t.Errorf("dir = %v %v", m, h)
	}
}

func TestInstallRejectsInvalidVersion(t *testing.T) {
	_, err := Install(context.Background(), server(t, "x", nil, ""), t.TempDir(), "../x", false, "linux", "amd64")
	if !errors.Is(err, godot.ErrInvalidGodotVersion) {
		t.Errorf("err = %v", err)
	}
}
