package godot

import (
	"errors"
	"strings"
	"testing"
)

const packageExample = `
[package]
name="My awesome package"
version="1.0.0"

[editor]
version=4.2.1

[assets]
custom_asset={ library="1234" }
other_asset={ version="optional-git-ref", git="https://github.com/other/asset.git" }

[dev_assets]
custom_dev_asset={ library="1709" }
other_dev_asset={ version="optional-git-ref", git="https://github.com/other/dev_asset.git" }

; alternative asset definition
[dev_assets.custom_other_dev_asset]
library="42"

[assets.custom_other_asset]
version="optional-git-ref"
git="https://github.com/other/custom_other_asset.git"
`

func TestProjectConfig(t *testing.T) {
	p, err := NewPackage([]byte(packageExample))
	if err != nil {
		t.Fatal(err)
	}
	if p.Package.Name != "My awesome package" || p.Editor.Version != "4.2.1" {
		t.Fatalf("got %+v", p)
	}
	if got := p.Assets["other_asset"].Git; got != "https://github.com/other/asset.git" {
		t.Fatal(got)
	}
	if got := p.DevAssets["custom_other_dev_asset"].Library; got != "42" {
		t.Fatal(got)
	}
	if len(p.Assets) != 3 || len(p.DevAssets) != 3 {
		t.Fatalf("assets %d dev %d", len(p.Assets), len(p.DevAssets))
	}
}

func TestNewPackageInvalidSource(t *testing.T) {
	if _, err := NewPackage([]byte("[package\nname=x")); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewPackageInvalidAssetValue(t *testing.T) {
	_, err := NewPackage([]byte("[assets]\nbroken=nonsense\n"))
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Fatalf("expected error naming asset, got %v", err)
	}
}

func TestAssetValidate(t *testing.T) {
	cases := []struct {
		a    PackageAssetConfiguration
		want error
	}{
		{PackageAssetConfiguration{Library: "1234"}, nil},
		{PackageAssetConfiguration{Git: "https://github.com/a/b", Version: "v1"}, nil},
		{PackageAssetConfiguration{}, ErrNoSource},
		{PackageAssetConfiguration{Library: "1", Git: "x", Version: "v"}, ErrMultipleSources},
		{PackageAssetConfiguration{Registry: "r", Version: "1"}, ErrRegistryUnsupported},
		{PackageAssetConfiguration{Library: "abc"}, ErrInvalidLibraryID},
		{PackageAssetConfiguration{Git: "https://github.com/a/b"}, ErrGitVersionRequired},
	}
	for _, c := range cases {
		if err := c.a.Validate(); !errors.Is(err, c.want) {
			t.Errorf("%+v: got %v, want %v", c.a, err, c.want)
		}
	}
}

func TestNewPackageValidatesAssets(t *testing.T) {
	_, err := NewPackage([]byte("[assets]\nfoo={ registry=\"r\", version=\"1\" }\n"))
	if !errors.Is(err, ErrRegistryUnsupported) || !strings.Contains(err.Error(), "foo") {
		t.Fatal(err)
	}
}

func TestNewPackageRejectsUnquotedInlineValue(t *testing.T) {
	for _, src := range []string{
		"[assets]\nfoo={ library=\"1\", version=2 }\n",
		"[assets]\nfoo={ version=2, library=\"1\" }\n",
	} {
		_, err := NewPackage([]byte(src))
		if !errors.Is(err, ErrInvalidAssetValue) || !strings.Contains(err.Error(), "foo") {
			t.Errorf("%q: got %v", src, err)
		}
	}
}

func TestNewPackageAcceptsEmptyQuotedValue(t *testing.T) {
	p, err := NewPackage([]byte("[assets]\nfoo={ library=\"1\", version=\"\" }\n"))
	if err != nil || p.Assets["foo"].Library != "1" {
		t.Fatal(p, err)
	}
}
