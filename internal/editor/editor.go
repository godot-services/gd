package editor

import (
	"errors"
	"fmt"
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

type editor struct {
	location string
	version  string
}

func NewEditor(location string) (*editor, error) {
	if !filepath.IsAbs(location) {
		return nil, errors.New("invalid location: is not absolute")
	}

	fileInfo, err := os.Stat(location)
	if errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if fileInfo.IsDir() {
		return nil, errors.New("invalid location: must be an executable")
	}

	cmd := exec.Command(location, EditorCmdFlagVersion)
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	version := string(stdout)
	return &editor{location, version}, nil
}
