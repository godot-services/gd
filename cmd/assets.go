package cmd

import (
	"context"
	"net/http"
	"os"
	"slices"

	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/install"
	"github.com/godot-services/gd/internal/source"
)

type options struct {
	libraryURL string
}

func (o *options) library() source.Library {
	return source.Library{BaseURL: o.libraryURL, Client: http.DefaultClient}
}

// installAsset downloads and extracts one asset. Without a resolved lock entry
// (nil or no URL) the asset is resolved first; a lock entry with SHA256 is verified.
func installAsset(ctx context.Context, lib source.Library, projectDir string, a godot.PackageAssetConfiguration, entry *godot.LockEntry) (godot.LockEntry, error) {
	var e godot.LockEntry
	if entry != nil {
		e = *entry
	}

	if e.URL == "" {
		r, err := source.Resolve(ctx, lib, a)
		if err != nil {
			return godot.LockEntry{}, err
		}
		e.Source, e.URL, e.VersionString = r.Source, r.URL, r.VersionString
	}

	tmp, err := os.MkdirTemp("", "gd-")
	if err != nil {
		return godot.LockEntry{}, err
	}
	defer os.RemoveAll(tmp)

	zipPath, sum, err := install.Fetch(ctx, lib.Client, e.URL, e.SHA256, tmp)
	if err != nil {
		return godot.LockEntry{}, err
	}

	paths, err := install.Extract(zipPath, projectDir, e.Paths)
	if err != nil {
		return godot.LockEntry{}, err
	}

	// folders the asset no longer ships would otherwise stay behind without an owner
	var dropped []string
	for _, p := range e.Paths {
		if !slices.Contains(paths, p) {
			dropped = append(dropped, p)
		}
	}
	if err := install.Remove(projectDir, dropped); err != nil {
		return godot.LockEntry{}, err
	}

	e.SHA256, e.Paths = sum, paths
	return e, nil
}

func loadProject() (dir string, pkg *godot.Package, lock godot.Lock, err error) {
	project, err := godot.NewProject(".")
	if err != nil {
		return "", nil, nil, err
	}
	if pkg, err = godot.LoadPackage(project.Location); err != nil {
		return "", nil, nil, err
	}
	if lock, err = godot.LoadLock(project.Location); err != nil {
		return "", nil, nil, err
	}
	return project.Location, pkg, lock, nil
}
