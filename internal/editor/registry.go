package editor

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/ini.v1"
)

const RegistryFileName = "editors.godot"

// aliasSection is reserved, so an editor cannot be named "aliases".
const aliasSection = "aliases"

var ErrReservedName = errors.New(`"aliases" is a reserved editor name`)

type Editor struct {
	Name, Path, Version string
	Dotnet              bool
	Build               string // official|custom
	Source              string // managed|linked
}

type Registry struct {
	Editors map[string]Editor
	Default string
}

// LoadRegistry reads editors.godot from dataDir; a missing file yields an empty registry.
func LoadRegistry(dataDir string) (*Registry, error) {
	r := &Registry{Editors: map[string]Editor{}}
	data, err := os.ReadFile(filepath.Join(dataDir, RegistryFileName))
	if errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	cfg, err := ini.Load(data)
	if err != nil {
		return nil, err
	}
	for _, s := range cfg.Sections() {
		switch s.Name() {
		case ini.DefaultSection:
		case aliasSection:
			r.Default = s.Key("default").String()
		default:
			dotnet, _ := s.Key("dotnet").Bool()
			r.Editors[s.Name()] = Editor{
				Name: s.Name(), Path: s.Key("path").String(), Version: s.Key("version").String(),
				Dotnet: dotnet, Build: s.Key("build").String(), Source: s.Key("source").String(),
			}
		}
	}
	return r, nil
}

// Save writes the registry to dataDir, creating it if needed.
func (r *Registry) Save(dataDir string) error {
	cfg := ini.Empty()
	if r.Default != "" {
		cfg.Section(aliasSection).Key("default").SetValue(r.Default)
	}
	names := make([]string, 0, len(r.Editors))
	for n := range r.Editors {
		if n == aliasSection {
			return ErrReservedName
		}
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		e, s := r.Editors[n], cfg.Section(n)
		s.Key("path").SetValue(e.Path)
		s.Key("version").SetValue(e.Version)
		s.Key("dotnet").SetValue(map[bool]string{true: "true", false: "false"}[e.Dotnet])
		s.Key("build").SetValue(e.Build)
		s.Key("source").SetValue(e.Source)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	return cfg.SaveTo(filepath.Join(dataDir, RegistryFileName))
}
