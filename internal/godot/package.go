package godot

import (
	"fmt"
	"maps"
	"os"
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
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	var config Package

	cfg.MapTo(&config)

	// 1️⃣ Standard mapping for "assets"-Section
	config.Assets = mapAssets(cfg.Section(packageAssetSectionName))

	// 2️⃣ Standard mapping for "dev_assets"-Section
	config.DevAssets = mapAssets(cfg.Section(packageDevAssetSectionName))

	// 3️⃣ Nested Sections like [assets.xxx] oder [dev_assets.xxx]
	nestedAssets, nestedDevAssets := nestedSections(cfg.Sections())
	maps.Copy(config.Assets, nestedAssets)
	maps.Copy(config.DevAssets, nestedDevAssets)

	return &config, nil
}

func mapAssets(section *ini.Section) map[string]PackageAssetConfiguration {
	assets := make(map[string]PackageAssetConfiguration)

	for _, key := range section.Keys() {
		asset, err := parseCustomINIValue(key.Value())
		if err != nil {
			fmt.Println("Fehler beim Parsen von", key.Name(), ":", err)
			continue
		}
		assets[key.Name()] = asset
	}

	return assets
}

func parseCustomINIValue(value string) (PackageAssetConfiguration, error) {
	asset := PackageAssetConfiguration{}

	// Regulärer Ausdruck für Schlüssel-Wert-Paare
	re := regexp.MustCompile(`(\w+)="([^"]+)"`)

	// Entferne geschweifte Klammern
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "{")
	value = strings.TrimSuffix(value, "}")

	matches := re.FindAllStringSubmatch(value, -1)
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
	dev_assets := make(map[string]PackageAssetConfiguration)

	for _, section := range sections {
		if strings.HasPrefix(section.Name(), nestedAssetsPrefix) || strings.HasPrefix(section.Name(), nestedDevAssetsPrefix) {
			asset := PackageAssetConfiguration{}
			asset.Version = section.Key("version").String()
			asset.Registry = section.Key("registry").String()
			asset.Git = section.Key("git").String()

			if strings.HasPrefix(section.Name(), nestedAssetsPrefix) {
				assets[strings.TrimPrefix(section.Name(), nestedAssetsPrefix)] = asset
			} else {
				dev_assets[strings.TrimPrefix(section.Name(), nestedDevAssetsPrefix)] = asset
			}
		}
	}

	return assets, dev_assets
}
