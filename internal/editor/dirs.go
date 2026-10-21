package editor

import (
	"os"
	"path/filepath"
	"runtime"
)

// DataDir is gd's own data directory (editors, templates, registry).
func DataDir() (string, error) {
	return dataDir("gd", "gd", "LOCALAPPDATA", "gd")
}

// GodotDataDir is the data directory the Godot editor itself uses (export templates live below it).
func GodotDataDir() (string, error) {
	return dataDir("godot", "Godot", "APPDATA", "Godot")
}

func dataDir(linux, mac, winEnv, win string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		if v := os.Getenv(winEnv); v != "" {
			return filepath.Join(v, win), nil
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", mac), nil
	default:
		if v := os.Getenv("XDG_DATA_HOME"); v != "" {
			return filepath.Join(v, linux), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", linux), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "AppData", map[string]string{"LOCALAPPDATA": "Local", "APPDATA": "Roaming"}[winEnv], win), nil
}
