package cmd

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/godot-services/gd/internal/editor"
	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/godotbuilds"
	"github.com/godot-services/gd/internal/install"
	"github.com/godot-services/gd/internal/source"
	"github.com/spf13/cobra"
)

type options struct {
	libraryURL   string
	buildsURL    string
	buildsAPIURL string
}

func (o *options) builds() godotbuilds.Client {
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	// no total timeout: export templates are >1 GB; dial (30s, default), TLS and header timeouts catch dead servers
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSHandshakeTimeout = 30 * time.Second
	tr.ResponseHeaderTimeout = 60 * time.Second
	return godotbuilds.Client{BaseURL: o.buildsURL, APIURL: o.buildsAPIURL, Token: token, HTTP: &http.Client{Transport: tr}}
}

func (o *options) library() source.Library {
	// timeout covers the whole download; generous for large assets on slow links
	return source.Library{BaseURL: o.libraryURL, Client: &http.Client{Timeout: 10 * time.Minute}}
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

	zipPath, sum, cleanup, err := fetch(ctx, lib, e)
	if err != nil {
		return godot.LockEntry{}, err
	}
	defer cleanup()

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

// installAndLock installs one asset and saves its lock entry right away, so a
// later failure keeps an accurate record of what is installed.
func installAndLock(ctx context.Context, lib source.Library, dir string, lock godot.Lock, name string, a godot.PackageAssetConfiguration, entry *godot.LockEntry) (godot.LockEntry, error) {
	installed, err := installAsset(ctx, lib, dir, a, entry)
	if err != nil {
		return godot.LockEntry{}, err
	}
	lock[name] = installed
	return installed, lock.Save(dir)
}

// cacheDir is where downloaded zips are cached (ADR 0014).
func cacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gd"), nil
}

// fetch uses the zip cache in <user cache dir>/gd; without a usable cache dir it
// downloads into a temp dir. cleanup removes temp downloads, cached zips stay.
func fetch(ctx context.Context, lib source.Library, e godot.LockEntry) (zipPath, sum string, cleanup func(), err error) {
	if cache, err := cacheDir(); err == nil {
		if err := os.MkdirAll(cache, 0o755); err == nil {
			zipPath, sum, err := install.FetchCached(ctx, lib.Client, e.URL, e.SHA256, cache)
			return zipPath, sum, func() {}, err
		}
	}

	tmp, err := os.MkdirTemp("", "gd-")
	if err != nil {
		return "", "", nil, err
	}
	cleanup = func() { os.RemoveAll(tmp) }
	zipPath, sum, err = install.Fetch(ctx, lib.Client, e.URL, e.SHA256, tmp)
	if err != nil {
		cleanup()
	}
	return zipPath, sum, cleanup, err
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

// warnVersionMismatch prints a warning to stderr when package.godot's editor version
// and project.godot's config/features differ; it never fails the command.
func warnVersionMismatch(cmd *cobra.Command, project *godot.Project, pkg *godot.Package) {
	if pkg.Editor.Version == "" {
		return
	}
	if v, err := project.GodotVersion(); err == nil {
		if w := editor.VersionWarning(v, pkg.Editor.Version); w != "" {
			cmd.PrintErrln(w)
		}
	}
}
