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

	ErrNoMainScene = errors.New("no main scene in project.godot")

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
	settings, err := LoadProjectSettings(p.Location)
	if err != nil {
		return "", err
	}

	raw, _ := settings.Get("application/config/features")
	features, _ := DecodeStringArray(raw)
	for _, f := range features {
		if godotVersionRe.MatchString(f) {
			return f, nil
		}
	}

	return "", ErrNoGodotVersion
}

// MainScene returns run/main_scene from project.godot, e.g. "res://main.tscn" or a uid:// reference.
func (p *Project) MainScene() (string, error) {
	settings, err := LoadProjectSettings(p.Location)
	if err != nil {
		return "", err
	}

	raw, _ := settings.Get("application/run/main_scene")
	if scene, _ := DecodeString(raw); scene != "" {
		return scene, nil
	}

	return "", ErrNoMainScene
}
