package editor

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/godotbuilds"
	"github.com/godot-services/gd/internal/install"
)

var ErrAlreadyInstalled = errors.New("editor already installed")

var errNoExecutable = errors.New("no Godot executable found")

// ExtractEditor unpacks an editor zip into target, keeping file modes and symlinks that stay inside target.
// All entries are checked before anything is written.
func ExtractEditor(zipPath, target string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()

	links := map[string]bool{}
	for _, f := range zr.File {
		name := path.Clean(strings.ReplaceAll(f.Name, "\\", "/"))
		if name == "." || path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			return fmt.Errorf("%w in zip: %s", install.ErrUnsafePath, f.Name)
		}
		// an entry below an archived symlink could make later ".." steps leave target
		for dir := path.Dir(name); dir != "."; dir = path.Dir(dir) {
			if links[dir] {
				return fmt.Errorf("%w in zip: %s below a symlink", install.ErrUnsafePath, f.Name)
			}
		}
		if f.Mode()&os.ModeSymlink == 0 {
			continue
		}
		dest, err := linkTarget(f)
		if err != nil {
			return err
		}
		if path.IsAbs(dest) || strings.HasPrefix(dest, "/") || strings.Contains(dest, ":") {
			return fmt.Errorf("%w in zip: absolute symlink %s", install.ErrUnsafePath, f.Name)
		}
		// ".." only as a leading run, so lexical and real resolution agree
		parts := strings.Split(dest, "/")
		for i, p := range parts {
			if p == ".." && i > 0 && parts[i-1] != ".." {
				return fmt.Errorf("%w in zip: symlink %s", install.ErrUnsafePath, f.Name)
			}
		}
		if r := path.Join(path.Dir(name), dest); r == ".." || strings.HasPrefix(r, "../") {
			return fmt.Errorf("%w in zip: symlink %s escapes", install.ErrUnsafePath, f.Name)
		}
		links[name] = true
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	real, err := filepath.EvalSymlinks(target)
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		dst := filepath.Join(real, filepath.FromSlash(path.Clean(strings.ReplaceAll(f.Name, "\\", "/"))))
		switch {
		case f.Mode()&os.ModeSymlink != 0:
			dest, _ := linkTarget(f)
			parent, err := ensureDir(real, filepath.Dir(dst))
			if err != nil {
				return err
			}
			lt := filepath.Join(parent, filepath.FromSlash(dest))
			if !within(real, lt) {
				return fmt.Errorf("%w in zip: symlink %s escapes", install.ErrUnsafePath, f.Name)
			}
			if r, err := filepath.EvalSymlinks(lt); err == nil && !within(real, r) {
				return fmt.Errorf("%w in zip: symlink %s escapes", install.ErrUnsafePath, f.Name)
			}
			if err := os.Symlink(dest, filepath.Join(parent, filepath.Base(dst))); err != nil {
				return err
			}
		case f.FileInfo().IsDir():
			if _, err := ensureDir(real, dst); err != nil {
				return err
			}
		default:
			parent, err := ensureDir(real, filepath.Dir(dst))
			if err != nil {
				return err
			}
			if err := writeEntry(f, filepath.Join(parent, filepath.Base(dst))); err != nil {
				return err
			}
		}
	}
	return nil
}

func within(real, p string) bool {
	return p == real || strings.HasPrefix(p, real+string(filepath.Separator))
}

// ensureDir creates dir and returns its resolved path, failing if any existing part resolves outside real.
// The real-path check, not entry names, is what holds on case-insensitive file systems.
func ensureDir(real, dir string) (string, error) {
	for anc := dir; ; anc = filepath.Dir(anc) {
		if r, err := filepath.EvalSymlinks(anc); err == nil {
			if !within(real, r) {
				return "", fmt.Errorf("%w: %s resolves outside the target", install.ErrUnsafePath, dir)
			}
			break
		}
		if anc == filepath.Dir(anc) {
			return "", fmt.Errorf("%w: %s", install.ErrUnsafePath, dir)
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	r, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	if !within(real, r) {
		return "", fmt.Errorf("%w: %s resolves outside the target", install.ErrUnsafePath, dir)
	}
	return r, nil
}

func linkTarget(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	b, err := io.ReadAll(io.LimitReader(rc, 4096))
	return strings.ReplaceAll(string(b), "\\", "/"), err
}

func writeEntry(f *zip.File, dst string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, f.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// FindExecutable locates the editor in an extracted tree (top level or one folder deep, as in .NET zips).
func FindExecutable(dir, goos string) (gui, console string, err error) {
	var pattern string
	switch goos {
	case "darwin":
		pattern = "*.app/Contents/MacOS/Godot"
	case "windows":
		pattern = "Godot_v*.exe"
	default:
		pattern = "Godot_v*"
	}
	for _, depth := range []string{"", "*"} {
		matches, _ := filepath.Glob(filepath.Join(dir, depth, pattern))
		for _, m := range matches {
			fi, err := os.Stat(m)
			if err != nil || !fi.Mode().IsRegular() {
				continue
			}
			base := filepath.Base(m)
			switch {
			case goos == "windows" && strings.HasSuffix(base, "_console.exe"):
				continue
			case goos != "windows" && goos != "darwin" && strings.HasSuffix(base, ".so"):
				continue
			}
			console = m
			if goos == "windows" {
				c := strings.TrimSuffix(m, ".exe") + "_console.exe"
				if _, err := os.Stat(c); err == nil {
					console = c
				}
			}
			return m, console, nil
		}
	}
	return "", "", fmt.Errorf("%w in %s", errNoExecutable, dir)
}

// Install downloads, verifies and unpacks an official editor into <dataDir>/editors/<name>.
// The caller registers the returned entry.
func Install(ctx context.Context, c godotbuilds.Client, dataDir, version string, dotnet bool, goos, goarch string) (Editor, error) {
	if err := godot.ValidateVersion(version); err != nil {
		return Editor{}, err
	}
	name := ManagedName(version, dotnet)
	editors := filepath.Join(dataDir, "editors")
	sweepStaging(editors)
	final := filepath.Join(editors, name)
	if _, err := os.Lstat(final); err == nil {
		return Editor{}, fmt.Errorf("%w: %s", ErrAlreadyInstalled, name)
	}

	tag := godot.ReleaseTag(version)
	sums, err := c.Sums(ctx, tag)
	if err != nil {
		return Editor{}, err
	}
	asset, err := godotbuilds.SelectEditorAsset(sums, goos, goarch, dotnet)
	if err != nil {
		return Editor{}, err
	}

	if err := os.MkdirAll(editors, 0o755); err != nil {
		return Editor{}, err
	}
	staging, err := os.MkdirTemp(editors, ".staging-")
	if err != nil {
		return Editor{}, err
	}
	defer os.RemoveAll(staging)

	zipPath, err := c.Download(ctx, tag, asset, sums[asset], staging)
	if err != nil {
		return Editor{}, err
	}
	tree := filepath.Join(staging, "tree")
	if err := ExtractEditor(zipPath, tree); err != nil {
		return Editor{}, err
	}
	gui, console, err := FindExecutable(tree, goos)
	if err != nil {
		return Editor{}, err
	}
	if goos == "linux" || goos == "darwin" {
		for _, p := range []string{gui, console} {
			if err := os.Chmod(p, 0o755); err != nil {
				return Editor{}, err
			}
		}
	}
	if err := os.Rename(tree, final); err != nil {
		return Editor{}, err
	}
	rel, _ := filepath.Rel(tree, gui)
	return Editor{Name: name, Path: filepath.Join(final, rel), Version: tag, Dotnet: dotnet, Build: "official", Source: "managed"}, nil
}

// ManagedName is the editors/ dir and registry name; 4.7.2-stable and 4.7.2 share one.
func ManagedName(version string, dotnet bool) string {
	name := strings.TrimSuffix(version, "-stable")
	if dotnet {
		name += "-dotnet"
	}
	return name
}

// sweepStaging removes staging dirs left by killed runs; the age keeps concurrent runs safe.
func sweepStaging(parent string) {
	entries, _ := os.ReadDir(parent)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".staging-") {
			continue
		}
		if fi, err := e.Info(); err == nil && time.Since(fi.ModTime()) > time.Hour {
			os.RemoveAll(filepath.Join(parent, e.Name()))
		}
	}
}
