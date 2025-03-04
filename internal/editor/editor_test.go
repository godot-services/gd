package editor_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/godot-services/gd/internal/editor"
	"github.com/godot-services/gd/internal/editor/testdata"
)

func TestNewEditorWithValidLocation(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	p := filepath.Join(cwd, "testdata", testdata.MockVersion)

	_, err = editor.NewEditor(p)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewEditorWithNotAbsoluteLocation(t *testing.T) {
	t.Parallel()

	p := filepath.Join("testdata", testdata.MockVersion)

	_, err := editor.NewEditor(p)
	if err != editor.ErrLocationMustBeAbs {
		t.Fatal("expected", editor.ErrLocationMustBeAbs)
	}
}

func TestNewEditorWithLocationToDir(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	p := filepath.Join(cwd, "testdata")

	_, err = editor.NewEditor(p)
	if err != editor.ErrLocationMustBeAFile {
		t.Fatal("expected", editor.ErrLocationMustBeAFile, "but got", err)
	}
}

func TestNewEditorWithUnknownLocation(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	p := filepath.Join(cwd, "unknown")

	_, err = editor.NewEditor(p)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatal("expected", os.ErrNotExist, "but got", err)
	}
}

func TestNewEditorWithLocationToNonExecutable(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	p := filepath.Join(cwd, "testdata", testdata.MockSimpleFile)

	_, err = editor.NewEditor(p)
	if err == nil {
		t.Fatal("expected execution error for non application but got no error")
	}
}
