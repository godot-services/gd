package godot

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

// Features:
// - manage installed versions
// - install additional versions
// - validate required depdencies to build from source
// - open project with specified editor
// - set default editor in system

const (
	EditorCmdFlagVersion = "--version"
)

var (
	ErrLocationMustBeAbs   = errors.New("invalid location: is not absolute")
	ErrLocationMustBeAFile = errors.New("invalid location: must be a file")
)

var shellGodotVersionCmd = func(location string) ([]byte, error) {
	return exec.Command(location, EditorCmdFlagVersion).Output()
}

type Editor struct {
	Location string
	Version  string
}

func NewEditor(location string) (*Editor, error) {
	if !filepath.IsAbs(location) {
		return nil, ErrLocationMustBeAbs
	}

	fileInfo, err := os.Stat(location)
	if errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if fileInfo.IsDir() {
		return nil, ErrLocationMustBeAFile
	}

	version, err := shellGodotVersionCmd(location)
	if err != nil {
		return nil, err
	}

	return &Editor{location, string(version)}, nil
}
