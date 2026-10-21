package godot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const (
	ProjectFileName = "project.godot"
)

var (
	ErrNoGodotProject = errors.New("no project.godot found")
	ErrNoGodotVersion = errors.New("no Godot version in project.godot, pass --godot")

	// only this line is read: project.godot may hold multi-line values ini.v1 cannot parse
	ErrNoMainScene = errors.New("no main scene in project.godot")

	mainSceneRe    = regexp.MustCompile(`(?m)^run/main_scene="([^"]*)"`)
	featuresRe     = regexp.MustCompile(`(?m)^config/features=PackedStringArray\(([^)]*)\)`)
	quotedRe       = regexp.MustCompile(`"([^"]*)"`)
	godotVersionRe = regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)
)

type Project struct {
	Location string
}

func NewProject(location string) (*Project, error) {
	normalizedLocation, err := normalizeLocation(location)
	if err != nil {
		return nil, err
	}

	if err := hasGodotProject(normalizedLocation); err != nil {
		return nil, err
	}

	return &Project{normalizedLocation}, nil
}

func hasGodotProject(location string) error {
	_, err := os.Stat(filepath.Join(location, ProjectFileName))
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrNoGodotProject, location)
	}

	return err
}

func normalizeLocation(location string) (string, error) {
	abs, err := filepath.Abs(location)
	if err != nil {
		return location, err
	}

	return locationDirectory(abs)
}

func locationDirectory(location string) (string, error) {
	fileInfo, err := os.Stat(location)
	if err != nil {
		return location, err
	}

	if fileInfo.IsDir() {
		return location, nil
	}

	return filepath.Dir(location), nil
}

// GodotVersion returns the engine version from config/features in project.godot, e.g. "4.3".
func (p *Project) GodotVersion() (string, error) {
	source, err := os.ReadFile(filepath.Join(p.Location, ProjectFileName))
	if err != nil {
		return "", err
	}

	if features := featuresRe.FindSubmatch(source); features != nil {
		for _, entry := range quotedRe.FindAllSubmatch(features[1], -1) {
			if godotVersionRe.Match(entry[1]) {
				return string(entry[1]), nil
			}
		}
	}

	return "", ErrNoGodotVersion
}

// MainScene returns run/main_scene from project.godot, e.g. "res://main.tscn" or a uid:// reference.
func (p *Project) MainScene() (string, error) {
	source, err := os.ReadFile(filepath.Join(p.Location, ProjectFileName))
	if err != nil {
		return "", err
	}

	if m := mainSceneRe.FindSubmatch(source); m != nil && len(m[1]) > 0 {
		return string(m[1]), nil
	}

	return "", ErrNoMainScene
}
