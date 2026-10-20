package godot

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/ini.v1"
)

const LocalFileName = "package.local.godot"

var (
	ErrInvalidGodotVersion = errors.New("invalid Godot version, expected 4.x[.y][-stable|-rcN|-betaN|-devN]")

	godotSpecRe  = regexp.MustCompile(`^4\.\d+(\.\d+)?(-(stable|rc\d+|beta\d+|dev\d+))?$`)
	majorMinorRe = regexp.MustCompile(`^\d+\.\d+`)
)

// ReleaseTag maps a plain version to its godot-builds tag: 4.7.2 -> 4.7.2-stable.
func ReleaseTag(version string) string {
	if strings.Contains(version, "-") {
		return version
	}
	return version + "-stable"
}

// MajorMinor returns "4.7" for 4.7.2, 4.7.2-stable and 4.7-rc1.
func MajorMinor(version string) string {
	return majorMinorRe.FindString(version)
}

// LoadLocalEditor returns [editor] name from package.local.godot; a missing file yields "".
func LoadLocalEditor(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, LocalFileName))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	cfg, err := ini.Load(data)
	if err != nil {
		return "", err
	}
	return cfg.Section("editor").Key("name").String(), nil
}

// SaveLocalEditor writes [editor] name to package.local.godot; an empty name removes the file.
func SaveLocalEditor(dir, name string) error {
	path := filepath.Join(dir, LocalFileName)
	if name == "" {
		if err := os.Remove(path); !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	cfg := ini.Empty()
	cfg.Section("editor").Key("name").SetValue(name)
	return cfg.SaveTo(path)
}
