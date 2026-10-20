package source

import "github.com/godot-services/gd/internal/godot"

func godotAsset(library, git, version string) godot.PackageAssetConfiguration {
	return godot.PackageAssetConfiguration{Library: library, Git: git, Version: version}
}
