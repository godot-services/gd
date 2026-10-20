package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/source"
	"github.com/spf13/cobra"
)

func newAddCmd(opts *options) *cobra.Command {
	var name, version string
	var dev bool

	cmd := &cobra.Command{
		Use:   "add <asset-id|git-url>",
		Short: "Add an asset from the asset library or a git repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := assetFromRef(args[0], version)
			if err != nil {
				return err
			}

			dir, pkg, lock, err := loadProject()
			if err != nil {
				return err
			}

			lib := opts.library()
			r, err := source.Resolve(cmd.Context(), lib, a)
			if err != nil {
				return err
			}

			if name == "" {
				name = source.Slug(r.Title)
			}
			if name == "" {
				return errors.New("cannot derive an asset name, use --name")
			}
			if _, ok := pkg.AllAssets()[name]; ok {
				return fmt.Errorf("%w: %s", godot.ErrAssetExists, name)
			}

			entry, err := installAsset(cmd.Context(), lib, dir, a, &godot.LockEntry{Source: r.Source, URL: r.URL, VersionString: r.VersionString})
			if err != nil {
				return err
			}

			if err := pkg.AddAsset(name, a, dev); err != nil {
				return err
			}
			if err := pkg.Save(dir); err != nil {
				return err
			}
			lock[name] = entry
			if err := lock.Save(dir); err != nil {
				return err
			}

			cmd.Printf("added %s (%s) -> %s\n", name, entry.VersionString, strings.Join(entry.Paths, ", "))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "asset name in package.godot (default: derived from the title)")
	cmd.Flags().StringVar(&version, "version", "", "git ref (tag, branch or commit); required for git URLs")
	cmd.Flags().BoolVar(&dev, "dev", false, "add as dev asset")

	return cmd
}

func assetFromRef(ref, version string) (godot.PackageAssetConfiguration, error) {
	var a godot.PackageAssetConfiguration

	switch {
	case ref != "" && strings.Trim(ref, "0123456789") == "":
		a.Library, a.Version = ref, version
	case strings.HasPrefix(ref, "https://") || strings.HasPrefix(ref, "http://"):
		a.Git, a.Version = ref, version
	default:
		return a, fmt.Errorf("expected an asset ID or a git URL, got %q", ref)
	}

	if err := a.Validate(); err != nil {
		if errors.Is(err, godot.ErrGitVersionRequired) {
			return a, fmt.Errorf("%w: pass --version", err)
		}
		return a, err
	}

	return a, nil
}
