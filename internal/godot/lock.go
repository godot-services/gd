package godot

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/ini.v1"
)

const LockFileName = "package.lock"

type LockEntry struct {
	Source        string
	URL           string
	SHA256        string
	VersionString string
	Paths         []string
}

// Lock maps asset names to their resolved download.
type Lock map[string]LockEntry

// LoadLock reads package.lock from dir; a missing file yields an empty lock.
func LoadLock(dir string) (Lock, error) {
	lock := Lock{}

	cfg, err := ini.Load(filepath.Join(dir, LockFileName))
	if errors.Is(err, os.ErrNotExist) {
		return lock, nil
	}
	if err != nil {
		return nil, err
	}

	for _, section := range cfg.Sections() {
		if section.Name() == ini.DefaultSection {
			continue
		}
		keys := section.KeysHash()
		entry := LockEntry{
			Source:        keys["source"],
			URL:           keys["url"],
			SHA256:        keys["sha256"],
			VersionString: keys["version_string"],
		}
		if keys["paths"] != "" {
			entry.Paths = strings.Split(keys["paths"], ",")
		}
		lock[section.Name()] = entry
	}

	return lock, nil
}

// Save writes package.lock to dir, sections sorted by name; an empty lock removes the file.
func (l Lock) Save(dir string) error {
	path := filepath.Join(dir, LockFileName)

	if len(l) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}

	cfg := ini.Empty()
	names := make([]string, 0, len(l))
	for name := range l {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		entry := l[name]
		section, err := cfg.NewSection(name)
		if err != nil {
			return err
		}
		for _, kv := range [][2]string{
			{"source", entry.Source},
			{"url", entry.URL},
			{"sha256", entry.SHA256},
			{"version_string", entry.VersionString},
			{"paths", strings.Join(entry.Paths, ",")},
		} {
			if _, err := section.NewKey(kv[0], kv[1]); err != nil {
				return err
			}
		}
	}

	return cfg.SaveTo(path)
}
