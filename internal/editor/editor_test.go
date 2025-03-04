package editor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/godot-services/gd/internal/editor"
	"github.com/godot-services/gd/internal/editor/testdata"
)

func TestNewEditor(t *testing.T) {
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
