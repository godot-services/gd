package godot

import (
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
custom_asset={ version="3.4.2", registry="my-custom-asset-registry" }
other_asset={ version="optional-git-ref", git="https://github.com/other/asset.git" }

[dev_assets]
custom_dev_asset={ version="3.4.2", registry="my-custom-asset-registry" }
other_dev_asset={ version="optional-git-ref", git="https://github.com/other/dev_asset.git" }

; alternative asset definition
[dev_assets.custom_other_dev_asset]
version="3.4.2"
registry="my-custom-asset-registry"

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
	if got := p.DevAssets["custom_other_dev_asset"].Registry; got != "my-custom-asset-registry" {
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
