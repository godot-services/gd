package cmd

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/install"
	"github.com/spf13/cobra"
)

func newUpdateCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "update [name...]",
		Short: "Update assets to their latest version (git assets keep their ref)",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, pkg, lock, err := loadProject()
			if err != nil {
				return err
			}

			assets := pkg.AllAssets()
			names := args
			if len(names) == 0 {
				names = slices.Collect(maps.Keys(assets))
			}
			for _, name := range names {
				if _, ok := assets[name]; !ok {
					return fmt.Errorf("%w: %s", godot.ErrAssetNotFound, name)
				}
			}
			slices.Sort(names)

			lib := opts.library()
			for _, name := range slices.Compact(names) {
				old := lock[name]
				// keep only the owned paths: forces a fresh resolve and allows replacing the old folders
				updated, err := installAndLock(cmd.Context(), lib, dir, lock, name, assets[name], &godot.LockEntry{Paths: old.Paths})
				if errors.Is(err, install.ErrForeignPath) {
					return fmt.Errorf("%w (if the asset was renamed in package.godot, run gd install first)", err)
				}
				if err != nil {
					return err
				}

				// same rule as gd outdated (URL or version string), plus new bytes behind the same URL
				if updated.URL != old.URL || updated.VersionString != old.VersionString || updated.SHA256 != old.SHA256 {
					oldVersion := old.VersionString
					if oldVersion == "" {
						oldVersion = "-"
					}
					cmd.Printf("updated %s %s -> %s\n", name, oldVersion, updated.VersionString)
				} else {
					cmd.Printf("%s is up to date\n", name)
				}
			}

			return nil
		},
	}
}
