// Package fakegodot lets a test binary stand in for the Godot executable.
package fakegodot

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Run must be the first call in TestMain. With GD_FAKE_GODOT=1 it behaves like
// Godot and exits; otherwise it returns false.
func Run() bool {
	if os.Getenv("GD_FAKE_GODOT") != "1" {
		return false
	}
	args := os.Args[1:]
	if log := os.Getenv("GD_FAKE_GODOT_LOG"); log != "" {
		if f, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintln(f, strings.Join(args, "\t"))
			f.Close()
		}
	}
	if len(args) > 0 && args[0] == "--version" {
		fmt.Println(os.Getenv("GD_FAKE_GODOT_VERSION"))
	}
	if d, err := time.ParseDuration(os.Getenv("GD_FAKE_GODOT_SLEEP")); err == nil {
		time.Sleep(d)
	}
	code, _ := strconv.Atoi(os.Getenv("GD_FAKE_GODOT_EXIT"))
	os.Exit(code)
	return true
}

// Setup enables fake mode for the test and returns the binary to run and the invocation log path.
func Setup(t *testing.T, version string) (binary, log string) {
	t.Helper()
	log = filepath.Join(t.TempDir(), "godot.log")
	t.Setenv("GD_FAKE_GODOT", "1")
	t.Setenv("GD_FAKE_GODOT_VERSION", version)
	t.Setenv("GD_FAKE_GODOT_LOG", log)
	t.Setenv("GD_FAKE_GODOT_SLEEP", "")
	t.Setenv("GD_FAKE_GODOT_EXIT", "")
	return os.Args[0], log
}
