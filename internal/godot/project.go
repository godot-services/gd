package godot

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	ProjectFileName = "project.godot"
)

type Project struct {
	Location string
}

func NewProject(location string) (*Project, error) {
	normalizedLocation, err := normalizeLocation(location)
	if err != nil {
		return nil, err
	}

	if has, err := hasGodotProject(normalizedLocation); !has || err != nil {
		return nil, err
	}

	return &Project{normalizedLocation}, nil
}

func hasGodotProject(location string) (bool, error) {
	_, err := os.Stat(filepath.Join(location, ProjectFileName))
	if errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	return true, nil
}

func normalizeLocation(location string) (string, error) {
	if filepath.IsAbs(location) {
		return locationDirectory(location)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return location, err
	}

	return locationDirectory(filepath.Join(cwd, location))
}

func locationDirectory(location string) (string, error) {
	fileInfo, err := os.Stat(location)
	if errors.Is(err, os.ErrNotExist) {
		return location, err
	}

	if fileInfo.IsDir() {
		return location, nil
	}

	return filepath.Dir(location), nil
}
