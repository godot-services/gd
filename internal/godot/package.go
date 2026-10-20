package godot

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/ini.v1"
)

const (
	PackageFileName = "package.godot"

	packageAssetSectionName    = "assets"
	packageDevAssetSectionName = "dev_assets"
	nestedAssetsPrefix         = "assets."
	nestedDevAssetsPrefix      = "dev_assets."
)

var (
	ErrInvalidAssetValue   = errors.New("invalid asset value")
	ErrNoSource            = errors.New("no source: set library or git")
	ErrMultipleSources     = errors.New("only one of library, git or registry may be set")
	ErrRegistryUnsupported = errors.New("registry sources are not supported yet")
	ErrInvalidLibraryID    = errors.New("library must be a numeric asset ID")
	ErrGitVersionRequired  = errors.New("git sources require a version")
	ErrAssetExists         = errors.New("asset already exists")
	ErrAssetNotFound       = errors.New("asset not found")

	assetValueRe = regexp.MustCompile(`(\w+)="([^"]+)"`)
)

type Package struct {
	Package   PackageConfiguration                 `ini:"package"`
	Editor    PackageEditorConfiguration           `ini:"editor"`
	Assets    map[string]PackageAssetConfiguration `ini:"assets"`
	DevAssets map[string]PackageAssetConfiguration `ini:"dev_assets"`

	file *ini.File
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
	Library  string
	Registry string
	Git      string
}

func (a PackageAssetConfiguration) Validate() error {
	sources := 0
	for _, s := range []string{a.Library, a.Git, a.Registry} {
		if s != "" {
			sources++
		}
	}

	switch {
	case sources == 0:
		return ErrNoSource
	case sources > 1:
		return ErrMultipleSources
	case a.Registry != "":
		return ErrRegistryUnsupported
	case a.Library != "" && strings.Trim(a.Library, "0123456789") != "":
		return ErrInvalidLibraryID
	case a.Git != "" && a.Version == "":
		return ErrGitVersionRequired
	}

	return nil
}

var packageLoadOptions = ini.LoadOptions{
	AllowNestedValues:      true,
	AllowNonUniqueSections: true,
}

// LoadPackage reads package.godot from dir; a missing file yields an empty package.
func LoadPackage(dir string) (*Package, error) {
	source, err := os.ReadFile(filepath.Join(dir, PackageFileName))
	if errors.Is(err, os.ErrNotExist) {
		return NewPackage(nil)
	}
	if err != nil {
		return nil, err
	}

	return NewPackage(source)
}

func NewPackage(source []byte) (*Package, error) {
	cfg, err := ini.LoadSources(packageLoadOptions, source)
	if err != nil {
		return nil, err
	}

	config := Package{file: cfg}

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

	for _, assets := range []map[string]PackageAssetConfiguration{config.Assets, config.DevAssets} {
		for name, asset := range assets {
			if err := asset.Validate(); err != nil {
				return nil, fmt.Errorf("asset %q: %w", name, err)
			}
		}
	}

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
		case "library":
			asset.Library = val
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
			// KeysHash instead of Key: Key creates missing keys, which Save would write back empty
			keys := section.KeysHash()
			asset := PackageAssetConfiguration{
				Version:  keys["version"],
				Library:  keys["library"],
				Registry: keys["registry"],
				Git:      keys["git"],
			}

			if strings.HasPrefix(section.Name(), nestedAssetsPrefix) {
				assets[strings.TrimPrefix(section.Name(), nestedAssetsPrefix)] = asset
			} else {
				devAssets[strings.TrimPrefix(section.Name(), nestedDevAssetsPrefix)] = asset
			}
		}
	}

	return assets, devAssets
}

// AllAssets returns assets and dev assets in one map.
func (p *Package) AllAssets() map[string]PackageAssetConfiguration {
	all := maps.Clone(p.Assets)
	maps.Copy(all, p.DevAssets)
	return all
}

func (p *Package) AddAsset(name string, a PackageAssetConfiguration, dev bool) error {
	if _, ok := p.AllAssets()[name]; ok {
		return fmt.Errorf("%w: %s", ErrAssetExists, name)
	}
	if err := a.Validate(); err != nil {
		return fmt.Errorf("asset %q: %w", name, err)
	}

	sectionName, assets := packageAssetSectionName, p.Assets
	if dev {
		sectionName, assets = packageDevAssetSectionName, p.DevAssets
	}

	if _, err := p.file.Section(sectionName).NewKey(name, inlineAssetValue(a)); err != nil {
		return err
	}
	assets[name] = a

	return nil
}

func (p *Package) RemoveAsset(name string) error {
	if _, ok := p.AllAssets()[name]; !ok {
		return fmt.Errorf("%w: %s", ErrAssetNotFound, name)
	}

	delete(p.Assets, name)
	delete(p.DevAssets, name)
	for _, sectionName := range []string{packageAssetSectionName, packageDevAssetSectionName} {
		p.file.Section(sectionName).DeleteKey(name)
		p.file.DeleteSection(sectionName + "." + name)
	}

	return nil
}

func (p *Package) Save(dir string) error {
	return p.file.SaveTo(filepath.Join(dir, PackageFileName))
}

func inlineAssetValue(a PackageAssetConfiguration) string {
	if a.Library != "" {
		return fmt.Sprintf(`{ library="%s" }`, a.Library)
	}
	return fmt.Sprintf(`{ git="%s", version="%s" }`, a.Git, a.Version)
}
