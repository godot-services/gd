// Package install downloads asset zips and extracts their addons into a project.
package install

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const addonsDir = "addons"

var (
	ErrHashMismatch = errors.New("sha256 mismatch")
	ErrNoAddons     = errors.New("asset contains no addons/ folder")
	ErrUnsafePath   = errors.New("unsafe path")
	ErrForeignPath  = errors.New("folder exists and is not managed by this asset")
)

// Fetch downloads url into tmpDir and returns the file path and its SHA-256.
// A non-empty wantSHA256 that does not match fails with ErrHashMismatch.
func Fetch(ctx context.Context, client *http.Client, url, wantSHA256, tmpDir string) (zipPath, gotSHA256 string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download %s: %s", url, resp.Status)
	}

	f, err := os.CreateTemp(tmpDir, "asset-*.zip")
	if err != nil {
		return "", "", err
	}
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(f.Name())
		}
	}()

	hash := sha256.New()
	if _, err = io.Copy(io.MultiWriter(f, hash), resp.Body); err != nil {
		return "", "", fmt.Errorf("download %s: %w", url, err)
	}
	if err = f.Close(); err != nil {
		return "", "", err
	}

	gotSHA256 = hex.EncodeToString(hash.Sum(nil))
	if wantSHA256 != "" && gotSHA256 != wantSHA256 {
		err = fmt.Errorf("%w for %s: got %s, want %s", ErrHashMismatch, url, gotSHA256, wantSHA256)
		return "", "", err
	}

	return f.Name(), gotSHA256, nil
}

// Extract installs addons/<x>/** of the zip into projectDir and returns the
// installed addons/<x> paths, sorted. Existing folders are only replaced when
// listed in owned. Nothing is written when any check fails.
func Extract(zipPath, projectDir string, owned []string) ([]string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	strip := commonTopLevel(r.File)

	type entry struct {
		file *zip.File
		name string
	}
	var entries []entry
	var tops []string

	for _, f := range r.File {
		name := path.Clean(strings.TrimPrefix(strings.ReplaceAll(f.Name, `\`, "/"), strip))
		if path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			return nil, fmt.Errorf("%w in zip: %s", ErrUnsafePath, f.Name)
		}

		parts := strings.Split(name, "/")
		// only files below addons/<x>/
		if f.FileInfo().IsDir() || len(parts) < 3 || parts[0] != addonsDir {
			continue
		}

		entries = append(entries, entry{f, name})
		if top := path.Join(parts[0], parts[1]); !slices.Contains(tops, top) {
			tops = append(tops, top)
		}
	}

	if len(entries) == 0 {
		return nil, ErrNoAddons
	}
	slices.Sort(tops)

	for _, top := range tops {
		_, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(top)))
		if err == nil && !slices.Contains(owned, top) {
			return nil, fmt.Errorf("%w: %s", ErrForeignPath, top)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	staging, err := os.MkdirTemp(projectDir, ".gd-staging-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(staging)

	for _, e := range entries {
		if err := extractFile(e.file, filepath.Join(staging, filepath.FromSlash(e.name))); err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(filepath.Join(projectDir, addonsDir), 0o755); err != nil {
		return nil, err
	}
	for _, top := range tops {
		target := filepath.Join(projectDir, filepath.FromSlash(top))
		if err := os.RemoveAll(target); err != nil {
			return nil, err
		}
		if err := os.Rename(filepath.Join(staging, filepath.FromSlash(top)), target); err != nil {
			return nil, err
		}
	}

	return tops, nil
}

// Remove deletes the given addons/<x> paths from projectDir; missing paths are ignored.
func Remove(projectDir string, paths []string) error {
	for _, p := range paths {
		clean := path.Clean(p)
		if clean != p || !strings.HasPrefix(clean, addonsDir+"/") || strings.Contains(clean, "..") {
			return fmt.Errorf("%w: %s", ErrUnsafePath, p)
		}
	}

	for _, p := range paths {
		if err := os.RemoveAll(filepath.Join(projectDir, filepath.FromSlash(p))); err != nil {
			return err
		}
	}

	return nil
}

// commonTopLevel returns "<dir>/" when every entry lives below one top-level
// folder other than addons (GitHub/GitLab archives), else "".
func commonTopLevel(files []*zip.File) string {
	top := ""
	for _, f := range files {
		first, _, found := strings.Cut(strings.ReplaceAll(f.Name, `\`, "/"), "/")
		if !found || (top != "" && first != top) {
			return ""
		}
		top = first
	}
	if top == "" || top == addonsDir {
		return ""
	}
	return top + "/"
}

// shortcut: no size limit per entry (zip bombs), add one if untrusted registries are supported
func extractFile(f *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}
