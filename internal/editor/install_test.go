package editor

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/godot-services/gd/internal/godotbuilds"
	"github.com/godot-services/gd/internal/install"
)

type zent struct {
	name, body string
	mode       os.FileMode
}

func makeZip(t *testing.T, ents ...zent) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range ents {
		h := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		h.SetMode(e.mode)
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(e.body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// server serves SHA512-SUMS.txt for 4.7.2-stable; sumOverride replaces the listed hash of asset.
func server(t *testing.T, asset string, zipData []byte, sumOverride string) godotbuilds.Client {
	t.Helper()
	sum := sha512.Sum512(zipData)
	h := hex.EncodeToString(sum[:])
	if sumOverride != "" {
		h = sumOverride
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/4.7.2-stable/SHA512-SUMS.txt":
			w.Write([]byte(h + "  " + asset + "\n"))
		case "/4.7.2-stable/" + asset:
			w.Write(zipData)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return godotbuilds.Client{BaseURL: srv.URL, HTTP: srv.Client()}
}

func isExec(t *testing.T, p string) {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o100 == 0 {
		t.Errorf("%s not executable", p)
	}
}

func TestInstallLinux(t *testing.T) {
	asset := "Godot_v4.7.2-stable_linux.x86_64.zip"
	c := server(t, asset, makeZip(t, zent{"Godot_v4.7.2-stable_linux.x86_64", "bin", 0o755}), "")
	dd := t.TempDir()
	ed, err := Install(context.Background(), c, dd, "4.7.2", false, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dd, "editors", "4.7.2", "Godot_v4.7.2-stable_linux.x86_64")
	if ed.Name != "4.7.2" || ed.Version != "4.7.2-stable" || ed.Path != want || ed.Build != "official" || ed.Source != "managed" || ed.Dotnet {
		t.Errorf("editor = %+v", ed)
	}
	isExec(t, ed.Path)
	if m, _ := filepath.Glob(filepath.Join(dd, "editors", ".staging-*")); len(m) != 0 {
		t.Errorf("staging left: %v", m)
	}
	if _, err := Install(context.Background(), c, dd, "4.7.2", false, "linux", "amd64"); !errors.Is(err, ErrAlreadyInstalled) {
		t.Errorf("second install err = %v", err)
	}
}

func TestInstallMacOSApp(t *testing.T) {
	asset := "Godot_v4.7.2-stable_macos.universal.zip"
	c := server(t, asset, makeZip(t,
		zent{"Godot.app/Contents/MacOS/Godot", "bin", 0o755},
		zent{"Godot.app/Contents/Frameworks/lib", "x", 0o644},
		zent{"Godot.app/Contents/Current", "Frameworks", os.ModeSymlink | 0o777},
	), "")
	dd := t.TempDir()
	ed, err := Install(context.Background(), c, dd, "4.7.2", false, "darwin", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dd, "editors", "4.7.2", "Godot.app", "Contents", "MacOS", "Godot")
	if ed.Path != want {
		t.Errorf("path = %s", ed.Path)
	}
	isExec(t, ed.Path)
	l := filepath.Join(dd, "editors", "4.7.2", "Godot.app", "Contents", "Current")
	if got, err := os.Readlink(l); err != nil || got != "Frameworks" {
		t.Errorf("symlink = %q, %v", got, err)
	}
}

func TestInstallWindows(t *testing.T) {
	asset := "Godot_v4.7.2-stable_win64.exe.zip"
	c := server(t, asset, makeZip(t,
		zent{"Godot_v4.7.2-stable_win64.exe", "g", 0o644},
		zent{"Godot_v4.7.2-stable_win64_console.exe", "c", 0o644},
	), "")
	dd := t.TempDir()
	ed, err := Install(context.Background(), c, dd, "4.7.2", false, "windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(ed.Path) != "Godot_v4.7.2-stable_win64.exe" {
		t.Errorf("path = %s", ed.Path)
	}
	dir := filepath.Join(dd, "editors", "4.7.2")
	gui, con, err := FindExecutable(dir, "windows")
	if err != nil || filepath.Base(gui) != "Godot_v4.7.2-stable_win64.exe" || filepath.Base(con) != "Godot_v4.7.2-stable_win64_console.exe" {
		t.Errorf("gui=%s console=%s err=%v", gui, con, err)
	}
}

func TestInstallDotnetFolder(t *testing.T) {
	asset := "Godot_v4.7.2-stable_mono_linux_x86_64.zip"
	c := server(t, asset, makeZip(t,
		zent{"Godot_v4.7.2-stable_mono_linux_x86_64/Godot_v4.7.2-stable_mono_linux.x86_64", "bin", 0o755},
		zent{"Godot_v4.7.2-stable_mono_linux_x86_64/GodotSharp/Api.dll", "dll", 0o644},
	), "")
	dd := t.TempDir()
	ed, err := Install(context.Background(), c, dd, "4.7.2", true, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if ed.Name != "4.7.2-dotnet" || !ed.Dotnet {
		t.Errorf("editor = %+v", ed)
	}
	isExec(t, ed.Path)
	if _, err := os.Stat(filepath.Join(dd, "editors", "4.7.2-dotnet", "Godot_v4.7.2-stable_mono_linux_x86_64", "GodotSharp", "Api.dll")); err != nil {
		t.Error(err)
	}
}

func writeZip(t *testing.T, ents ...zent) (zipPath, target string) {
	t.Helper()
	zipPath = filepath.Join(t.TempDir(), "e.zip")
	if err := os.WriteFile(zipPath, makeZip(t, ents...), 0o644); err != nil {
		t.Fatal(err)
	}
	return zipPath, filepath.Join(t.TempDir(), "out")
}

func TestExtractEditorRejectsUnsafe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks")
	}
	cases := map[string][]zent{
		"symlink escape":   {{"ok.txt", "x", 0o644}, {"a/link", "../../outside", os.ModeSymlink | 0o777}},
		"absolute symlink": {{"a/link", "/etc/passwd", os.ModeSymlink | 0o777}},
		"dotdot entry":     {{"../evil", "x", 0o644}},
		"absolute entry":   {{"/evil", "x", 0o644}},
	}
	for name, ents := range cases {
		t.Run(name, func(t *testing.T) {
			z, target := writeZip(t, ents...)
			if err := ExtractEditor(z, target); !errors.Is(err, install.ErrUnsafePath) {
				t.Fatalf("err = %v", err)
			}
			if _, err := os.Stat(filepath.Join(target, "ok.txt")); err == nil {
				t.Error("wrote files from a bad archive")
			}
		})
	}
}

func TestInstallHashMismatchLeavesNothing(t *testing.T) {
	asset := "Godot_v4.7.2-stable_linux.x86_64.zip"
	c := server(t, asset, makeZip(t, zent{"Godot_v4.7.2-stable_linux.x86_64", "bin", 0o755}), "deadbeef")
	dd := t.TempDir()
	if _, err := Install(context.Background(), c, dd, "4.7.2", false, "linux", "amd64"); err == nil {
		t.Fatal("expected error")
	}
	ents, _ := os.ReadDir(filepath.Join(dd, "editors"))
	if len(ents) != 0 {
		t.Errorf("left behind: %v", ents)
	}
}

func caseInsensitive(t *testing.T) bool {
	t.Helper()
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "a"), nil, 0o644)
	_, err := os.Stat(filepath.Join(d, "A"))
	return err == nil
}

func TestExtractEditorRejectsCaseFoldedSymlinkChain(t *testing.T) {
	if !caseInsensitive(t) {
		t.Skip("case-sensitive file system")
	}
	root := t.TempDir()
	zp := filepath.Join(root, "e.zip")
	os.WriteFile(zp, makeZip(t,
		zent{"L", ".", os.ModeSymlink | 0o777},
		zent{"l/M", ".", os.ModeSymlink | 0o777},
		zent{"l/m/s", "../../X", os.ModeSymlink | 0o777},
		zent{"l/m/S/evil", "x", 0o644},
	), 0o644)
	os.MkdirAll(filepath.Join(root, "a", "X"), 0o755)
	target := filepath.Join(root, "a", "b", "out")
	if err := ExtractEditor(zp, target); !errors.Is(err, install.ErrUnsafePath) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "X", "evil")); err == nil {
		t.Fatal("evil written outside target")
	}
}

func TestExtractEditorRejectsDotEntry(t *testing.T) {
	for _, name := range []string{".", "./"} {
		z, target := writeZip(t,
			zent{name, ".", os.ModeSymlink | 0o777},
			zent{"evil", "x", 0o644},
		)
		if err := ExtractEditor(z, target); !errors.Is(err, install.ErrUnsafePath) {
			t.Errorf("%q: err = %v", name, err)
		}
	}
}

func TestInstallMakesBinaryExecutable(t *testing.T) {
	asset := "Godot_v4.7.2-stable_linux.x86_64.zip"
	c := server(t, asset, makeZip(t, zent{"Godot_v4.7.2-stable_linux.x86_64", "bin", 0o644}), "")
	ed, err := Install(context.Background(), c, t.TempDir(), "4.7.2", false, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	isExec(t, ed.Path)
}
