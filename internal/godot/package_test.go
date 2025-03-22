package godot

import (
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
	_, err := NewPackage([]byte(packageExample))
	if err != nil {
		t.Fatal(err)
	}
}
