package godot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectConfig(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	p := filepath.Join(cwd, "testdata", "packages_example.godot")

	_, err = NewProjectConfiguration(p)
	if err != nil {
		t.Fatal(err)
	}
}
