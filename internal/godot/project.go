package godot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ProjectFileName = "project.godot"
)

var ErrNoGodotProject = errors.New("no project.godot found")

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
