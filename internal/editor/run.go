package editor

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ConsolePath returns the Windows console twin of a GUI executable
// (Godot_..._win64.exe -> Godot_..._win64_console.exe) when it exists, else gui.
func ConsolePath(gui string) string {
	console := strings.TrimSuffix(gui, ".exe") + "_console.exe"
	if _, err := os.Stat(console); err == nil {
		return console
	}
	return gui
}

// Run executes path and returns its exit code; err is set only when it could not start.
func Run(ctx context.Context, path string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	c := exec.CommandContext(ctx, path, args...)
	c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr
	err := c.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	return 0, err
}

// StartDetached starts path in its own session without waiting for it.
func StartDetached(path string, args []string) error {
	c := exec.Command(path, args...)
	c.SysProcAttr = detachAttr()
	if err := c.Start(); err != nil {
		return err
	}
	return c.Process.Release()
}
