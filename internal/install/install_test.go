package install

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func makeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "asset.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write([]byte(content))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestExtractStripsTopLevelAndKeepsAddonsOnly(t *testing.T) {
	project := t.TempDir()
	z := makeZip(t, map[string]string{
		"repo-main/addons/x/a.gd": "a",
		"repo-main/demo/b.gd":     "b",
		"repo-main/README.md":     "r",
	})

	paths, err := Extract(z, project, nil)
	if err != nil || !reflect.DeepEqual(paths, []string{"addons/x"}) {
		t.Fatal(paths, err)
	}
	if readFile(t, filepath.Join(project, "addons", "x", "a.gd")) != "a" {
		t.Fatal("file content")
	}
	for _, p := range []string{"demo", "README.md", "repo-main"} {
		if _, err := os.Stat(filepath.Join(project, p)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s should not exist", p)
		}
	}
}

func TestExtractWithoutTopLevelFolder(t *testing.T) {
	project := t.TempDir()
	z := makeZip(t, map[string]string{"addons/x/a.gd": "a", "addons/y/b.gd": "b"})

	paths, err := Extract(z, project, nil)
	if err != nil || !reflect.DeepEqual(paths, []string{"addons/x", "addons/y"}) {
		t.Fatal(paths, err)
	}
}

func TestExtractNoAddons(t *testing.T) {
	z := makeZip(t, map[string]string{"repo/demo/a.gd": "a"})
	if _, err := Extract(z, t.TempDir(), nil); !errors.Is(err, ErrNoAddons) {
		t.Fatal(err)
	}
}

func TestExtractZipSlip(t *testing.T) {
	project := t.TempDir()
	z := makeZip(t, map[string]string{
		"repo/addons/x/ok.gd":       "ok",
		"repo/addons/../../evil.gd": "evil",
	})

	if _, err := Extract(z, project, nil); !errors.Is(err, ErrUnsafePath) {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(project); len(entries) != 0 {
		t.Fatalf("project not empty: %v", entries)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(project), "evil.gd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("evil.gd written outside project")
	}
}

func TestExtractForeignFolder(t *testing.T) {
	project := t.TempDir()
	os.MkdirAll(filepath.Join(project, "addons", "x"), 0o755)
	os.WriteFile(filepath.Join(project, "addons", "x", "mine.gd"), []byte("mine"), 0o644)
	z := makeZip(t, map[string]string{"repo/addons/x/a.gd": "a"})

	if _, err := Extract(z, project, nil); !errors.Is(err, ErrForeignPath) {
		t.Fatal(err)
	}
	if readFile(t, filepath.Join(project, "addons", "x", "mine.gd")) != "mine" {
		t.Fatal("foreign file changed")
	}
}

func TestExtractReplacesOwnedFolder(t *testing.T) {
	project := t.TempDir()
	os.MkdirAll(filepath.Join(project, "addons", "x"), 0o755)
	os.WriteFile(filepath.Join(project, "addons", "x", "old.gd"), []byte("old"), 0o644)
	z := makeZip(t, map[string]string{"repo/addons/x/new.gd": "new"})

	if _, err := Extract(z, project, []string{"addons/x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "addons", "x", "old.gd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("old file still there")
	}
	if readFile(t, filepath.Join(project, "addons", "x", "new.gd")) != "new" {
		t.Fatal("new file missing")
	}
	if entries, _ := os.ReadDir(project); len(entries) != 1 {
		t.Fatalf("staging left behind: %v", entries)
	}
}

func serve(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchVerifiesHash(t *testing.T) {
	srv := serve(t, 200, "zipdata")
	sum := sha256.Sum256([]byte("zipdata"))
	want := hex.EncodeToString(sum[:])

	path, got, err := Fetch(context.Background(), srv.Client(), srv.URL, "", t.TempDir())
	if err != nil || got != want || readFile(t, path) != "zipdata" {
		t.Fatal(path, got, err)
	}

	if _, _, err := Fetch(context.Background(), srv.Client(), srv.URL, "deadbeef", t.TempDir()); !errors.Is(err, ErrHashMismatch) {
		t.Fatal(err)
	}
}

func TestFetchHTTPError(t *testing.T) {
	srv := serve(t, 500, "boom")
	tmp := t.TempDir()

	if _, _, err := Fetch(context.Background(), srv.Client(), srv.URL, "", tmp); err == nil {
		t.Fatal("expected error")
	}
	if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
		t.Fatalf("left behind: %v", entries)
	}
}

func TestRemoveRejectsOutsideAddons(t *testing.T) {
	for _, p := range []string{"../x", "addons/../x", "x", "/etc", "addons"} {
		if err := Remove(t.TempDir(), []string{p}); !errors.Is(err, ErrUnsafePath) {
			t.Errorf("%s: %v", p, err)
		}
	}
}

func TestRemoveDeletesPaths(t *testing.T) {
	project := t.TempDir()
	os.MkdirAll(filepath.Join(project, "addons", "x"), 0o755)
	os.MkdirAll(filepath.Join(project, "addons", "y"), 0o755)

	if err := Remove(project, []string{"addons/x", "addons/missing"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "addons", "x")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("not removed")
	}
	if _, err := os.Stat(filepath.Join(project, "addons", "y")); err != nil {
		t.Fatal("removed too much")
	}
}

func TestRemoveFolderWithDotsInName(t *testing.T) {
	project := t.TempDir()
	paths, err := Extract(makeZip(t, map[string]string{"repo/addons/my..tool/a.gd": "a"}), project, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := Remove(project, paths); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "addons", "my..tool")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("not removed")
	}
}

func TestExtractRejectsCommaInFolderName(t *testing.T) {
	project := t.TempDir()
	z := makeZip(t, map[string]string{"repo/addons/a,b/a.gd": "a"})

	if _, err := Extract(z, project, nil); !errors.Is(err, ErrUnsafePath) {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(project); len(entries) != 0 {
		t.Fatalf("project not empty: %v", entries)
	}
}
