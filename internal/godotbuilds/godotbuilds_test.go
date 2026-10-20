package godotbuilds

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/godot-services/gd/internal/install"
)

// real asset names of the godot-builds releases
const (
	assets472 = `
godot-4.7.2-stable.tar.xz
godot-4.7.2-stable.tar.xz.sha256
godot-lib.4.7.2.stable.mono.template_release.aar
godot-lib.4.7.2.stable.template_release.aar
godot-lib.4.7.2.stable.template_release.perfetto.aar
Godot_native_debug_symbols.4.7.2.stable.editor.android.zip
Godot_native_debug_symbols.4.7.2.stable.template_release.android.zip
Godot_v4.7.2-stable_android_debug.perfetto.apk
Godot_v4.7.2-stable_android_editor.aab
Godot_v4.7.2-stable_android_editor.apk
Godot_v4.7.2-stable_android_editor_horizonos.apk
Godot_v4.7.2-stable_android_editor_picoos.apk
Godot_v4.7.2-stable_android_release.perfetto.apk
Godot_v4.7.2-stable_android_source.perfetto.zip
Godot_v4.7.2-stable_export_templates.tpz
Godot_v4.7.2-stable_linux.arm32.zip
Godot_v4.7.2-stable_linux.arm64.zip
Godot_v4.7.2-stable_linux.x86_32.zip
Godot_v4.7.2-stable_linux.x86_64.zip
Godot_v4.7.2-stable_macos.universal.zip
Godot_v4.7.2-stable_mono_export_templates.tpz
Godot_v4.7.2-stable_mono_linux_arm32.zip
Godot_v4.7.2-stable_mono_linux_arm64.zip
Godot_v4.7.2-stable_mono_linux_x86_32.zip
Godot_v4.7.2-stable_mono_linux_x86_64.zip
Godot_v4.7.2-stable_mono_macos.universal.zip
Godot_v4.7.2-stable_mono_win32.zip
Godot_v4.7.2-stable_mono_win64.zip
Godot_v4.7.2-stable_mono_windows_arm64.zip
Godot_v4.7.2-stable_web_editor.zip
Godot_v4.7.2-stable_win32.exe.zip
Godot_v4.7.2-stable_win64.exe.zip
Godot_v4.7.2-stable_windows_arm64.exe.zip
SHA512-SUMS.txt
`
	assets48b1 = `
godot-4.8-beta1.tar.xz
godot-4.8-beta1.tar.xz.sha256
godot-lib.4.8.beta1.mono.template_release.aar
godot-lib.4.8.beta1.template_release.aar
godot-lib.4.8.beta1.template_release.perfetto.aar
Godot_v4.8-beta1_android_debug.perfetto.apk
Godot_v4.8-beta1_android_editor.aab
Godot_v4.8-beta1_android_editor.apk
Godot_v4.8-beta1_android_editor.debugsymbols.zip
Godot_v4.8-beta1_android_editor_horizonos.apk
Godot_v4.8-beta1_android_editor_picoos.apk
Godot_v4.8-beta1_android_release.perfetto.apk
Godot_v4.8-beta1_android_source.perfetto.zip
Godot_v4.8-beta1_debugsymbols_android_release.zip
Godot_v4.8-beta1_debugsymbols_linux_debug.arm32.zip
Godot_v4.8-beta1_debugsymbols_linux_debug.arm64.zip
Godot_v4.8-beta1_debugsymbols_linux_debug.x86_32.zip
Godot_v4.8-beta1_debugsymbols_linux_debug.x86_64.zip
Godot_v4.8-beta1_debugsymbols_linux_release.arm32.zip
Godot_v4.8-beta1_debugsymbols_linux_release.arm64.zip
Godot_v4.8-beta1_debugsymbols_linux_release.x86_32.zip
Godot_v4.8-beta1_debugsymbols_linux_release.x86_64.zip
Godot_v4.8-beta1_debugsymbols_macos_debug.arm64.zip
Godot_v4.8-beta1_debugsymbols_macos_debug.x86_64.zip
Godot_v4.8-beta1_debugsymbols_macos_release.arm64.zip
Godot_v4.8-beta1_debugsymbols_macos_release.x86_64.zip
Godot_v4.8-beta1_debugsymbols_windows_debug_arm64.exe.zip
Godot_v4.8-beta1_debugsymbols_windows_debug_x86_32.exe.zip
Godot_v4.8-beta1_debugsymbols_windows_debug_x86_64.exe.zip
Godot_v4.8-beta1_debugsymbols_windows_release_arm64.exe.zip
Godot_v4.8-beta1_debugsymbols_windows_release_x86_32.exe.zip
Godot_v4.8-beta1_debugsymbols_windows_release_x86_64.exe.zip
Godot_v4.8-beta1_export_templates.tpz
Godot_v4.8-beta1_linux.arm32.debugsymbols.zip
Godot_v4.8-beta1_linux.arm32.zip
Godot_v4.8-beta1_linux.arm64.debugsymbols.zip
Godot_v4.8-beta1_linux.arm64.zip
Godot_v4.8-beta1_linux.x86_32.debugsymbols.zip
Godot_v4.8-beta1_linux.x86_32.zip
Godot_v4.8-beta1_linux.x86_64.debugsymbols.zip
Godot_v4.8-beta1_linux.x86_64.zip
Godot_v4.8-beta1_macos.arm64.dSYM.zip
Godot_v4.8-beta1_macos.arm64.zip
Godot_v4.8-beta1_macos.x86_64.dSYM.zip
Godot_v4.8-beta1_macos.x86_64.zip
Godot_v4.8-beta1_mono_debugsymbols_linux_debug.arm32.zip
Godot_v4.8-beta1_mono_debugsymbols_linux_debug.arm64.zip
Godot_v4.8-beta1_mono_debugsymbols_linux_debug.x86_32.zip
Godot_v4.8-beta1_mono_debugsymbols_linux_debug.x86_64.zip
Godot_v4.8-beta1_mono_debugsymbols_linux_release.arm32.zip
Godot_v4.8-beta1_mono_debugsymbols_linux_release.arm64.zip
Godot_v4.8-beta1_mono_debugsymbols_linux_release.x86_32.zip
Godot_v4.8-beta1_mono_debugsymbols_linux_release.x86_64.zip
Godot_v4.8-beta1_mono_debugsymbols_macos_debug.arm64.zip
Godot_v4.8-beta1_mono_debugsymbols_macos_debug.x86_64.zip
Godot_v4.8-beta1_mono_debugsymbols_macos_release.arm64.zip
Godot_v4.8-beta1_mono_debugsymbols_macos_release.x86_64.zip
Godot_v4.8-beta1_mono_debugsymbols_windows_debug_arm64.exe.zip
Godot_v4.8-beta1_mono_debugsymbols_windows_debug_x86_32.exe.zip
Godot_v4.8-beta1_mono_debugsymbols_windows_debug_x86_64.exe.zip
Godot_v4.8-beta1_mono_debugsymbols_windows_release_arm64.exe.zip
Godot_v4.8-beta1_mono_debugsymbols_windows_release_x86_32.exe.zip
Godot_v4.8-beta1_mono_debugsymbols_windows_release_x86_64.exe.zip
Godot_v4.8-beta1_mono_export_templates.tpz
Godot_v4.8-beta1_mono_linux.arm32.debugsymbols.zip
Godot_v4.8-beta1_mono_linux.arm64.debugsymbols.zip
Godot_v4.8-beta1_mono_linux.x86_32.debugsymbols.zip
Godot_v4.8-beta1_mono_linux.x86_64.debugsymbols.zip
Godot_v4.8-beta1_mono_linux_arm32.zip
Godot_v4.8-beta1_mono_linux_arm64.zip
Godot_v4.8-beta1_mono_linux_x86_32.zip
Godot_v4.8-beta1_mono_linux_x86_64.zip
Godot_v4.8-beta1_mono_macos.arm64.dSYM.zip
Godot_v4.8-beta1_mono_macos.arm64.zip
Godot_v4.8-beta1_mono_macos.x86_64.dSYM.zip
Godot_v4.8-beta1_mono_macos.x86_64.zip
Godot_v4.8-beta1_mono_win32.exe.debugsymbols.zip
Godot_v4.8-beta1_mono_win32.zip
Godot_v4.8-beta1_mono_win64.exe.debugsymbols.zip
Godot_v4.8-beta1_mono_win64.zip
Godot_v4.8-beta1_mono_windows_arm64.exe.debugsymbols.zip
Godot_v4.8-beta1_mono_windows_arm64.zip
Godot_v4.8-beta1_web_editor.zip
Godot_v4.8-beta1_win32.exe.debugsymbols.zip
Godot_v4.8-beta1_win32.exe.zip
Godot_v4.8-beta1_win64.exe.debugsymbols.zip
Godot_v4.8-beta1_win64.exe.zip
Godot_v4.8-beta1_windows_arm64.exe.debugsymbols.zip
Godot_v4.8-beta1_windows_arm64.exe.zip
SHA512-SUMS.txt
`
)

func sumsOf(names string) map[string]string {
	m := map[string]string{}
	for _, n := range strings.Fields(names) {
		m[n] = "sum-" + n
	}
	return m
}

func TestSelectEditorAsset(t *testing.T) {
	tests := []struct {
		release, goos, goarch string
		dotnet                bool
		want                  string
	}{
		{"472", "linux", "amd64", false, "Godot_v4.7.2-stable_linux.x86_64.zip"},
		{"472", "linux", "arm64", false, "Godot_v4.7.2-stable_linux.arm64.zip"},
		{"472", "linux", "amd64", true, "Godot_v4.7.2-stable_mono_linux_x86_64.zip"},
		{"472", "linux", "arm64", true, "Godot_v4.7.2-stable_mono_linux_arm64.zip"},
		{"472", "darwin", "arm64", false, "Godot_v4.7.2-stable_macos.universal.zip"},
		{"472", "darwin", "amd64", true, "Godot_v4.7.2-stable_mono_macos.universal.zip"},
		{"472", "windows", "amd64", false, "Godot_v4.7.2-stable_win64.exe.zip"},
		{"472", "windows", "arm64", false, "Godot_v4.7.2-stable_windows_arm64.exe.zip"},
		{"472", "windows", "amd64", true, "Godot_v4.7.2-stable_mono_win64.zip"},
		{"472", "windows", "arm64", true, "Godot_v4.7.2-stable_mono_windows_arm64.zip"},
		{"48", "linux", "amd64", false, "Godot_v4.8-beta1_linux.x86_64.zip"},
		{"48", "linux", "arm64", true, "Godot_v4.8-beta1_mono_linux_arm64.zip"},
		{"48", "darwin", "arm64", false, "Godot_v4.8-beta1_macos.arm64.zip"},
		{"48", "darwin", "amd64", false, "Godot_v4.8-beta1_macos.x86_64.zip"},
		{"48", "darwin", "arm64", true, "Godot_v4.8-beta1_mono_macos.arm64.zip"},
		{"48", "darwin", "amd64", true, "Godot_v4.8-beta1_mono_macos.x86_64.zip"},
		{"48", "windows", "amd64", false, "Godot_v4.8-beta1_win64.exe.zip"},
		{"48", "windows", "arm64", false, "Godot_v4.8-beta1_windows_arm64.exe.zip"},
		{"48", "windows", "amd64", true, "Godot_v4.8-beta1_mono_win64.zip"},
		{"48", "windows", "arm64", true, "Godot_v4.8-beta1_mono_windows_arm64.zip"},
	}
	sums := map[string]map[string]string{"472": sumsOf(assets472), "48": sumsOf(assets48b1)}
	for _, tt := range tests {
		got, err := SelectEditorAsset(sums[tt.release], tt.goos, tt.goarch, tt.dotnet)
		if err != nil || got != tt.want {
			t.Errorf("%s/%s dotnet=%v: got %q, %v; want %q", tt.goos, tt.goarch, tt.dotnet, got, err, tt.want)
		}
	}
}

func TestSelectEditorAssetMissing(t *testing.T) {
	sums := map[string]string{
		"Godot_v4.7.2-stable_win64.exe.zip":                 "a",
		"Godot_v4.7.2-stable_linux.x86_64.zip":              "b",
		"Godot_v4.7.2-stable_linux.x86_64.debugsymbols.zip": "c",
		"SHA512-SUMS.txt":                                   "d",
	}
	_, err := SelectEditorAsset(sums, "linux", "arm64", false)
	if !errors.Is(err, ErrNoAsset) {
		t.Fatalf("err = %v, want ErrNoAsset", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "linux/arm64") ||
		!strings.Contains(msg, "Godot_v4.7.2-stable_linux.x86_64.zip, Godot_v4.7.2-stable_win64.exe.zip") ||
		strings.Contains(msg, "debugsymbols") {
		t.Errorf("message = %q", msg)
	}
}

func TestTemplatesAsset(t *testing.T) {
	if got := TemplatesAsset("4.7.2-stable", false); got != "Godot_v4.7.2-stable_export_templates.tpz" {
		t.Errorf("got %q", got)
	}
	if got := TemplatesAsset("4.7.2-stable", true); got != "Godot_v4.7.2-stable_mono_export_templates.tpz" {
		t.Errorf("got %q", got)
	}
}

func TestSums(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/4.7.2-stable/SHA512-SUMS.txt":
			w.Write([]byte("abc123  Godot_v4.7.2-stable_win64.exe.zip\n\ndef456  Godot_v4.7.2-stable_linux.x86_64.zip\n"))
		case "/boom/SHA512-SUMS.txt":
			http.Error(w, "x", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}

	got, err := c.Sums(context.Background(), "4.7.2-stable")
	if err != nil || len(got) != 2 || got["Godot_v4.7.2-stable_win64.exe.zip"] != "abc123" || got["Godot_v4.7.2-stable_linux.x86_64.zip"] != "def456" {
		t.Fatalf("got %v, %v", got, err)
	}
	if _, err := c.Sums(context.Background(), "9.9-stable"); !errors.Is(err, ErrUnknownVersion) {
		t.Errorf("404: err = %v, want ErrUnknownVersion", err)
	}
	if _, err := c.Sums(context.Background(), "boom"); err == nil || errors.Is(err, ErrUnknownVersion) || !strings.Contains(err.Error(), "500") {
		t.Errorf("500: err = %v", err)
	}
}

func TestDownloadVerifiesSHA512(t *testing.T) {
	body := []byte("editor bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/4.7.2-stable/a.zip" {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	defer srv.Close()
	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}

	sum := sha512.Sum512(body)
	path, err := c.Download(context.Background(), "4.7.2-stable", "a.zip", hex.EncodeToString(sum[:]), t.TempDir())
	if err != nil || path == "" {
		t.Fatalf("got %q, %v", path, err)
	}

	dir := t.TempDir()
	if _, err := c.Download(context.Background(), "4.7.2-stable", "a.zip", strings.Repeat("0", 128), dir); !errors.Is(err, install.ErrHashMismatch) {
		t.Errorf("err = %v, want ErrHashMismatch", err)
	}
}
