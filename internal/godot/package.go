package godot

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"

	"gopkg.in/ini.v1"
)

const (
	packageAssetSectionName    = "assets"
	packageDevAssetSectionName = "dev_assets"
	nestedAssetsPrefix         = "assets."
	nestedDevAssetsPrefix      = "dev_assets."
)

var (
	ErrInvalidAssetValue = errors.New("invalid asset value")
	assetValueRe         = regexp.MustCompile(`(\w+)="([^"]+)"`)
)

type Package struct {
	Package   PackageConfiguration                 `ini:"package"`
	Editor    PackageEditorConfiguration           `ini:"editor"`
	Assets    map[string]PackageAssetConfiguration `ini:"assets"`
	DevAssets map[string]PackageAssetConfiguration `ini:"dev_assets"`
}

type PackageConfiguration struct {
	Name    string `ini:"name"`
	Version string `ini:"version"`
	License string `ini:"license"`
}

type PackageEditorConfiguration struct {
	Version string `ini:"version"`
}

type PackageAssetConfiguration struct {
	Version  string
	Registry string
	Git      string
}

func NewPackage(source []byte) (*Package, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowNestedValues:      true,
		AllowNonUniqueSections: true,
	}, source)
	if err != nil {
		return nil, err
	}

	var config Package

	if err := cfg.MapTo(&config); err != nil {
		return nil, err
	}

	// 1️⃣ Standard mapping for "assets"-Section
	if config.Assets, err = mapAssets(cfg.Section(packageAssetSectionName)); err != nil {
		return nil, err
	}

	// 2️⃣ Standard mapping for "dev_assets"-Section
	if config.DevAssets, err = mapAssets(cfg.Section(packageDevAssetSectionName)); err != nil {
		return nil, err
	}

	// 3️⃣ Nested Sections like [assets.xxx] or [dev_assets.xxx]
	nestedAssets, nestedDevAssets := nestedSections(cfg.Sections())
	maps.Copy(config.Assets, nestedAssets)
	maps.Copy(config.DevAssets, nestedDevAssets)

	return &config, nil
}

func mapAssets(section *ini.Section) (map[string]PackageAssetConfiguration, error) {
	assets := make(map[string]PackageAssetConfiguration)

	for _, key := range section.Keys() {
		asset, err := parseCustomINIValue(key.Value())
		if err != nil {
			return nil, fmt.Errorf("asset %q: %w", key.Name(), err)
		}
		assets[key.Name()] = asset
	}

	return assets, nil
}

func parseCustomINIValue(value string) (PackageAssetConfiguration, error) {
	asset := PackageAssetConfiguration{}

	// strip surrounding braces of the inline table
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "{")
	value = strings.TrimSuffix(value, "}")

	matches := assetValueRe.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return asset, fmt.Errorf("%w: %s", ErrInvalidAssetValue, value)
	}
	for _, match := range matches {
		key, val := match[1], match[2]
		switch key {
		case "version":
			asset.Version = val
		case "registry":
			asset.Registry = val
		case "git":
			asset.Git = val
		}
	}

	return asset, nil
}

func nestedSections(sections []*ini.Section) (map[string]PackageAssetConfiguration, map[string]PackageAssetConfiguration) {
	assets := make(map[string]PackageAssetConfiguration)
	devAssets := make(map[string]PackageAssetConfiguration)

	for _, section := range sections {
		if strings.HasPrefix(section.Name(), nestedAssetsPrefix) || strings.HasPrefix(section.Name(), nestedDevAssetsPrefix) {
			asset := PackageAssetConfiguration{}
			asset.Version = section.Key("version").String()
			asset.Registry = section.Key("registry").String()
			asset.Git = section.Key("git").String()

			if strings.HasPrefix(section.Name(), nestedAssetsPrefix) {
				assets[strings.TrimPrefix(section.Name(), nestedAssetsPrefix)] = asset
			} else {
				devAssets[strings.TrimPrefix(section.Name(), nestedDevAssetsPrefix)] = asset
			}
		}
	}

	return assets, devAssets
}
