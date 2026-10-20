package godot

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePackageFile(t *testing.T, dir, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, PackageFileName), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPackageRoundTrip(t *testing.T) {
	dir := t.TempDir()
	writePackageFile(t, dir, "; my comment\n[package]\nname=\"Game\"\n\n[assets]\nfoo={ library=\"1\" }\n")

	p, err := LoadPackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.AddAsset("bar", PackageAssetConfiguration{Git: "https://github.com/a/b", Version: "v1"}, true); err != nil {
		t.Fatal(err)
	}
	if err := p.Save(dir); err != nil {
		t.Fatal(err)
	}

	out, _ := os.ReadFile(filepath.Join(dir, PackageFileName))
	if !strings.Contains(string(out), "my comment") {
		t.Fatalf("comment lost:\n%s", out)
	}

	q, err := LoadPackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if q.Assets["foo"].Library != "1" || q.DevAssets["bar"].Version != "v1" || q.DevAssets["bar"].Git != "https://github.com/a/b" || q.Package.Name != "Game" {
		t.Fatalf("got %+v\n%s", q, out)
	}
}

func TestPackageRemoveNestedAsset(t *testing.T) {
	dir := t.TempDir()
	writePackageFile(t, dir, "[assets]\nkeep={ library=\"2\" }\n\n[assets.foo]\nlibrary=\"1\"\n")

	p, _ := LoadPackage(dir)
	if err := p.RemoveAsset("foo"); err != nil {
		t.Fatal(err)
	}
	if err := p.Save(dir); err != nil {
		t.Fatal(err)
	}

	q, err := LoadPackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := q.Assets["foo"]; ok || q.Assets["keep"].Library != "2" {
		t.Fatalf("got %+v", q.Assets)
	}
}

func TestPackageRemoveInlineAsset(t *testing.T) {
	dir := t.TempDir()
	writePackageFile(t, dir, "[dev_assets]\nfoo={ library=\"1\" }\n")

	p, _ := LoadPackage(dir)
	if err := p.RemoveAsset("foo"); err != nil {
		t.Fatal(err)
	}
	p.Save(dir)

	q, _ := LoadPackage(dir)
	if len(q.DevAssets) != 0 {
		t.Fatalf("got %+v", q.DevAssets)
	}
	if err := q.RemoveAsset("foo"); !errors.Is(err, ErrAssetNotFound) {
		t.Fatal(err)
	}
}

func TestPackageAddExisting(t *testing.T) {
	p, _ := LoadPackage(t.TempDir())
	a := PackageAssetConfiguration{Library: "1"}
	if err := p.AddAsset("foo", a, false); err != nil {
		t.Fatal(err)
	}
	if err := p.AddAsset("foo", a, true); !errors.Is(err, ErrAssetExists) {
		t.Fatal(err)
	}
	if err := p.AddAsset("bar", PackageAssetConfiguration{}, false); !errors.Is(err, ErrNoSource) {
		t.Fatal(err)
	}
}

func TestLoadPackageMissingFile(t *testing.T) {
	dir := t.TempDir()
	p, err := LoadPackage(dir)
	if err != nil || len(p.Assets) != 0 || len(p.DevAssets) != 0 {
		t.Fatal(p, err)
	}
	if err := p.AddAsset("foo", PackageAssetConfiguration{Library: "1"}, false); err != nil {
		t.Fatal(err)
	}
	if err := p.Save(dir); err != nil {
		t.Fatal(err)
	}
	q, err := LoadPackage(dir)
	if err != nil || q.Assets["foo"].Library != "1" {
		t.Fatal(q, err)
	}
	if all := q.AllAssets(); len(all) != 1 {
		t.Fatal(all)
	}
}

func TestPackageSaveKeepsNestedSectionUnchanged(t *testing.T) {
	dir := t.TempDir()
	writePackageFile(t, dir, "[assets.foo]\ngit=\"https://github.com/a/b\"\nversion=\"v1\"\n")

	p, _ := LoadPackage(dir)
	if err := p.Save(dir); err != nil {
		t.Fatal(err)
	}

	out, _ := os.ReadFile(filepath.Join(dir, PackageFileName))
	if strings.Contains(string(out), "library") || strings.Contains(string(out), "registry") {
		t.Fatalf("empty keys added:\n%s", out)
	}
}

func TestPackageDuplicateSectionsAreMerged(t *testing.T) {
	p, err := NewPackage([]byte("[assets]\na={ library=\"1\" }\n\n[dev_assets]\n\n[assets]\nc={ library=\"2\" }\n"))
	if err != nil || p.Assets["a"].Library != "1" || p.Assets["c"].Library != "2" {
		t.Fatal(p, err)
	}
}

func TestPackageQuotedValuesWithCommentChars(t *testing.T) {
	dir := t.TempDir()
	writePackageFile(t, dir, "[package]\nname=\"A;B#C\"\n\n[assets]\nfoo={ git=\"https://github.com/a/b\", version=\"rel#1\" }\n")

	p, err := LoadPackage(dir)
	if err != nil || p.Package.Name != "A;B#C" || p.Assets["foo"].Version != "rel#1" {
		t.Fatal(p, err)
	}
	p.AddAsset("bar", PackageAssetConfiguration{Library: "2"}, false)
	if err := p.Save(dir); err != nil {
		t.Fatal(err)
	}

	q, err := LoadPackage(dir)
	if err != nil || q.Package.Name != "A;B#C" || q.Assets["foo"].Version != "rel#1" || q.Assets["bar"].Library != "2" {
		out, _ := os.ReadFile(filepath.Join(dir, PackageFileName))
		t.Fatalf("%+v %v\n%s", q, err, out)
	}
}
