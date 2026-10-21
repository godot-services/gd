package cmd

import (
	"os"
	"testing"

	"github.com/godot-services/gd/internal/testutil/fakegodot"
)

func TestMain(m *testing.M) {
	if fakegodot.Run() {
		return
	}
	os.Exit(m.Run())
}
