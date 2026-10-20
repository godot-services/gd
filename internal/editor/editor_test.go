package editor

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/godot-services/gd/internal/testutil/fakegodot"
)

func TestMain(m *testing.M) {
	if fakegodot.Run() {
		return
	}
	os.Exit(m.Run())
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in      string
		version string
		dotnet  bool
		build   string
	}{
		{"4.7.2.stable.official.abc", "4.7.2-stable", false, "official"},
		{"4.7.2.stable.mono.official.abc\n", "4.7.2-stable", true, "official"},
		{"4.7.2.stable.custom_build.abc", "4.7.2-stable", false, "custom"},
		{"4.8.beta1.official.x", "4.8-beta1", false, "official"},
	}
	for _, tt := range tests {
		v, d, b, err := ParseVersion(tt.in)
		if err != nil || v != tt.version || d != tt.dotnet || b != tt.build {
			t.Errorf("ParseVersion(%q) = %q,%v,%q,%v", tt.in, v, d, b, err)
		}
	}
	for _, bad := range []string{"", "garbage", "Godot Engine"} {
		if _, _, _, err := ParseVersion(bad); err == nil {
			t.Errorf("ParseVersion(%q): want error", bad)
		}
	}
}

func TestProbeVersion(t *testing.T) {
	bin, log := fakegodot.Setup(t, "4.7.2.stable.mono.official.abc")
	v, d, b, err := ProbeVersion(context.Background(), bin)
	if err != nil || v != "4.7.2-stable" || !d || b != "official" {
		t.Fatalf("got %q,%v,%q,%v", v, d, b, err)
	}
	if got, _ := os.ReadFile(log); strings.TrimSpace(string(got)) != "--version" {
		t.Errorf("log = %q", got)
	}
	t.Setenv("GD_FAKE_GODOT_EXIT", "3")
	if _, _, _, err := ProbeVersion(context.Background(), bin); err == nil {
		t.Error("want error on non-zero exit")
	}
}

func TestRegistryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	r, err := LoadRegistry(dir)
	if err != nil || len(r.Editors) != 0 || r.Default != "" {
		t.Fatalf("empty load: %+v %v", r, err)
	}
	r.Default = "4.7.2"
	r.Editors["4.7.2"] = Editor{Name: "4.7.2", Path: "/a/godot", Version: "4.7.2-stable", Build: "official", Source: "managed"}
	r.Editors["mine"] = Editor{Name: "mine", Path: "/b/godot", Version: "4.8-beta1", Dotnet: true, Build: "custom", Source: "linked"}
	if err := r.Save(dir); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRegistry(dir)
	if err != nil || !reflect.DeepEqual(got, r) {
		t.Fatalf("got %+v, %v", got, err)
	}
	r.Editors["aliases"] = Editor{Name: "aliases"}
	if err := r.Save(dir); err == nil {
		t.Error("want reserved name error")
	}
}

func TestDataDirs(t *testing.T) {
	t.Setenv("HOME", "/h")
	t.Setenv("XDG_DATA_HOME", "/x")
	t.Setenv("LOCALAPPDATA", "/l")
	t.Setenv("APPDATA", "/a")
	var wantGD, wantGodot string
	switch runtime.GOOS {
	case "windows":
		wantGD, wantGodot = filepath.Join("/l", "gd"), filepath.Join("/a", "Godot")
	case "darwin":
		wantGD, wantGodot = "/h/Library/Application Support/gd", "/h/Library/Application Support/Godot"
	default:
		wantGD, wantGodot = "/x/gd", "/x/godot"
		t.Setenv("XDG_DATA_HOME", "")
		if d, _ := DataDir(); d != "/h/.local/share/gd" {
			t.Errorf("fallback DataDir = %q", d)
		}
		t.Setenv("XDG_DATA_HOME", "/x")
	}
	if d, _ := DataDir(); d != wantGD {
		t.Errorf("DataDir = %q, want %q", d, wantGD)
	}
	if d, _ := GodotDataDir(); d != wantGodot {
		t.Errorf("GodotDataDir = %q, want %q", d, wantGodot)
	}
}
