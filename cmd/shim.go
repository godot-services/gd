package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/godot-services/gd/internal/editor"
	"github.com/spf13/cobra"
)

func newEditorShimCmd() *cobra.Command {
	c := &cobra.Command{Use: "shim", Short: "Manage a godot command for your PATH that runs gd godot"}
	c.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Write <gd data>/bin/godot; gd never changes PATH itself",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := shimTarget()
			if err != nil {
				return err
			}
			path, err := shimPathFor(runtime.GOOS)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			_, content := shimFile(runtime.GOOS, exe)
			if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
				return err
			}
			if err := os.Chmod(path, 0o755); err != nil {
				return err
			}
			cmd.Printf("installed %s -> %s\n", path, exe)
			if dir := filepath.Dir(path); !onPath(dir) {
				if runtime.GOOS == "windows" {
					cmd.Printf("add %s to your PATH (user environment variables)\n", dir)
				} else {
					cmd.Printf("add %s to your PATH, for example in your shell profile:\n  export PATH=\"%s:$PATH\"\n", dir, dir)
				}
			}
			return nil
		},
	}, &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the godot shim",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := shimPathFor(runtime.GOOS)
			if err != nil {
				return err
			}
			if err := os.Remove(path); errors.Is(err, os.ErrNotExist) {
				cmd.Println("no shim installed")
				return nil
			} else if err != nil {
				return err
			}
			cmd.Printf("removed %s\n", path)
			return nil
		},
	})
	return c
}

// shimTarget is the gd the shim runs: the gd on PATH when that is this binary,
// so a package manager's bin/gd link survives upgrades (os.Executable resolves
// links on Linux), else this binary.
func shimTarget() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if p, err := exec.LookPath("gd"); err == nil {
		if p, err = filepath.Abs(p); err == nil && sameFile(p, exe) {
			return p, nil
		}
	}
	return exe, nil
}

func sameFile(a, b string) bool {
	fa, err := os.Stat(a)
	if err != nil {
		return false
	}
	fb, err := os.Stat(b)
	return err == nil && os.SameFile(fa, fb)
}

func shimPathFor(goos string) (string, error) {
	dataDir, err := editor.DataDir()
	if err != nil {
		return "", err
	}
	name, _ := shimFile(goos, "")
	return filepath.Join(dataDir, "bin", name), nil
}

// shimFile is the shim's file name and content: it runs gd at exe, or gd from
// PATH when exe is gone (ADR 0021).
func shimFile(goos, exe string) (name, content string) {
	if goos == "windows" {
		exe = `"` + strings.ReplaceAll(exe, "%", "%%") + `"`
		// goto instead of ( ) blocks: paths like "Program Files (x86)" would end a block
		return "godot.cmd", strings.Join([]string{
			"@echo off",
			"rem written by gd editor shim install",
			"if not exist " + exe + " goto path",
			exe + " godot -- %*",
			"exit /b",
			":path",
			"gd godot -- %*",
			"",
		}, "\r\n")
	}
	exe = "'" + strings.ReplaceAll(exe, "'", `'\''`) + "'"
	return "godot", "#!/bin/sh\n" +
		"# written by gd editor shim install\n" +
		"if [ -x " + exe + " ]; then exec " + exe + " godot -- \"$@\"; fi\n" +
		"exec gd godot -- \"$@\"\n"
}

func onPath(dir string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p != "" && (filepath.Clean(p) == filepath.Clean(dir) || runtime.GOOS == "windows" && strings.EqualFold(filepath.Clean(p), filepath.Clean(dir))) {
			return true
		}
	}
	return false
}
