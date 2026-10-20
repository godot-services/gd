package editor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/godotbuilds"
)

var (
	ErrTemplatesInstalled = errors.New("templates already installed")
	ErrUnknownTemplates   = errors.New("no such export templates")
)

// the leading alnum keeps ".." out; version.txt becomes a directory name
var templatesVersionRe = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z.]*$`)

// TemplatesDir is where Godot looks for export templates below its data dir.
func TemplatesDir(godotDataDir string) string {
	return filepath.Join(godotDataDir, "export_templates")
}

// SelfContainedTemplatesDir returns the templates dir of a self-contained editor
// (`._sc_` or `_sc_` next to the binary, or next to the .app on macOS).
func SelfContainedTemplatesDir(editorPath string) (string, bool) {
	dir := filepath.Dir(editorPath)
	if filepath.Base(dir) == "MacOS" && filepath.Base(filepath.Dir(dir)) == "Contents" {
		if app := filepath.Dir(filepath.Dir(dir)); strings.HasSuffix(app, ".app") {
			dir = filepath.Dir(app)
		}
	}
	for _, m := range []string{"._sc_", "_sc_"} {
		if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
			return filepath.Join(dir, "editor_data", "export_templates"), true
		}
	}
	return "", false
}

// templatesName is the directory name Godot uses for tag: 4.7.2-stable -> 4.7.2.stable[.mono].
func templatesName(tag string, dotnet bool) string {
	n := strings.Replace(tag, "-", ".", 1)
	if dotnet {
		n += ".mono"
	}
	return n
}

// InstallTemplates downloads and verifies the .tpz of tag and installs it below
// templatesDir under the name from its version.txt. It returns the install path.
func InstallTemplates(ctx context.Context, c godotbuilds.Client, tag string, dotnet bool, templatesDir string, force bool) (string, error) {
	expected := templatesName(tag, dotnet)
	if err := godot.ValidateVersion(tag); err != nil {
		return "", err
	}
	sweepStaging(templatesDir)
	if _, err := os.Lstat(filepath.Join(templatesDir, expected)); err == nil && !force {
		return "", fmt.Errorf("%w: %s", ErrTemplatesInstalled, expected)
	}
	sums, err := c.Sums(ctx, tag)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(templatesDir, ".staging-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)

	asset := godotbuilds.TemplatesAsset(tag, dotnet)
	tpzPath, err := c.Download(ctx, tag, asset, sums[asset], staging)
	if err != nil {
		return "", err
	}
	tree := filepath.Join(staging, "tree")
	if err := ExtractEditor(tpzPath, tree); err != nil {
		return "", err
	}
	raw, err := os.ReadFile(filepath.Join(tree, "templates", "version.txt"))
	if err != nil {
		return "", fmt.Errorf("%s: %w", asset, err)
	}
	version := strings.TrimSpace(string(raw))
	if version != expected {
		return "", fmt.Errorf("%s: templates/version.txt is %q, expected %q", asset, version, expected)
	}
	final := filepath.Join(templatesDir, expected)
	// the old dir moves into staging (removed by the defer) so a failed swap can restore it
	oldDir := filepath.Join(staging, "old")
	_, statErr := os.Lstat(final)
	hadOld := statErr == nil
	if hadOld {
		if !force {
			return "", fmt.Errorf("%w: %s", ErrTemplatesInstalled, expected)
		}
		if err := os.Rename(final, oldDir); err != nil {
			return "", err
		}
	}
	if err := os.Rename(filepath.Join(tree, "templates"), final); err != nil {
		if hadOld {
			os.Rename(oldDir, final)
		}
		return "", err
	}
	return final, nil
}

// ListTemplates returns the installed template directory names, sorted.
func ListTemplates(templatesDir string) ([]string, error) {
	entries, err := os.ReadDir(templatesDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)
	return names, nil
}

// UninstallTemplates removes the direct child version of templatesDir.
func UninstallTemplates(templatesDir, version string) error {
	if !templatesVersionRe.MatchString(version) {
		return fmt.Errorf("invalid templates version %q", version)
	}
	p := filepath.Join(templatesDir, version)
	if fi, err := os.Lstat(p); err != nil || !fi.IsDir() {
		return fmt.Errorf("%w %q", ErrUnknownTemplates, version)
	}
	return os.RemoveAll(p)
}
