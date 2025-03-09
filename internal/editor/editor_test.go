package editor

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewEditorWithLocationToExpectedExecutable(t *testing.T) {
	// mock shell execution
	origShellGodotVersionCmd := shellGodotVersionCmd
	defer func() { shellGodotVersionCmd = origShellGodotVersionCmd }()
	shellGodotVersionCmdCalled := false
	shellGodotVersionCmd = func(location string) ([]byte, error) {
		shellGodotVersionCmdCalled = true
		return []byte("example-godot-version"), nil
	}

	dummyAsExecutable, err := currentFilename()
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewEditor(dummyAsExecutable)
	if err != nil {
		t.Fatal(err)
	}

	if !shellGodotVersionCmdCalled {
		t.Fatal("expected shellGodotVersionCmdCalled to be true")
	}
}

func TestNewEditorWithNotAbsoluteLocation(t *testing.T) {
	t.Parallel()

	p := "example-godot-executable"

	_, err := NewEditor(p)
	if err != ErrLocationMustBeAbs {
		t.Fatal("expected", ErrLocationMustBeAbs)
	}
}

func TestNewEditorWithLocationToDir(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewEditor(cwd)
	if err != ErrLocationMustBeAFile {
		t.Fatal("expected", ErrLocationMustBeAFile, "but got", err)
	}
}

func TestNewEditorWithUnknownLocation(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	p := filepath.Join(cwd, "unknown")

	_, err = NewEditor(p)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatal("expected", os.ErrNotExist, "but got", err)
	}
}

func TestNewEditorWithLocationToNonExecutable(t *testing.T) {
	// mock shell execution
	origShellGodotVersionCmd := shellGodotVersionCmd
	defer func() { shellGodotVersionCmd = origShellGodotVersionCmd }()
	shellGodotVersionCmdCalled := false
	shellGodotVersionCmd = func(location string) ([]byte, error) {
		shellGodotVersionCmdCalled = true
		return nil, errors.New("expected error")
	}

	dummyAsNonExecutable, err := currentFilename()
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewEditor(dummyAsNonExecutable)
	if err == nil {
		t.Fatal("expected execution error for non application but got no error")
	}

	if !shellGodotVersionCmdCalled {
		t.Fatal("expected shellGodotVersionCmdCalled to be true")
	}
}

func currentFilename() (string, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", errors.New("unable to get the current filename as mock for an existing file")
	}
	return filename, nil
}
