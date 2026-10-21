package cmd

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/install"
	"github.com/godot-services/gd/internal/source"
	"github.com/spf13/cobra"
)

var errLockWithoutHash = errors.New("lock entry has no sha256, run gd remove and gd add again")

func newInstallCmd(opts *options) *cobra.Command {
	var noDev bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install all assets of package.godot as pinned in package-lock.godot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, pkg, lock, err := loadProject()
			if err != nil {
				return err
			}

			if project, err := godot.NewProject("."); err == nil {
				warnVersionMismatch(cmd, project, pkg)
			}

			lib := opts.library()
			known, assets := pkg.AllAssets(), pkg.AllAssets()
			if noDev {
				assets = pkg.Assets
			}

			// stale entries first: a renamed asset must not see its old folder as foreign.
			// Dev assets are never stale, also not with --no-dev.
			for name, entry := range lock {
				if _, ok := known[name]; ok {
					continue
				}
				if err := install.Remove(dir, entry.Paths); err != nil {
					return err
				}
				delete(lock, name)
				if err := lock.Save(dir); err != nil {
					return err
				}
				cmd.Printf("removed %s\n", name)
			}

			for _, name := range slices.Sorted(maps.Keys(assets)) {
				a := assets[name]
				var entry *godot.LockEntry
				if e, ok := lock[name]; ok {
					if e.Source != source.ID(a) {
						// source changed in package.godot: resolve again, keep ownership of installed paths
						e = godot.LockEntry{Paths: e.Paths}
					} else if e.URL != "" && e.SHA256 == "" {
						return fmt.Errorf("%s: %w", name, errLockWithoutHash)
					}
					entry = &e
				}

				installed, err := installAndLock(cmd.Context(), lib, dir, lock, name, a, entry)
				if err != nil {
					return err
				}
				cmd.Printf("installed %s (%s) -> %s\n", name, installed.VersionString, strings.Join(installed.Paths, ", "))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&noDev, "no-dev", false, "skip dev_assets (they are neither installed nor removed)")

	return cmd
}
