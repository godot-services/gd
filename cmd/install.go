package cmd

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/install"
	"github.com/godot-services/gd/internal/source"
	"github.com/spf13/cobra"
)

var errLockWithoutHash = errors.New("lock entry has no sha256, run gd remove and gd add again")

func newInstallCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install all assets of package.godot as pinned in package-lock.godot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, pkg, lock, err := loadProject()
			if err != nil {
				return err
			}

			lib := opts.library()
			assets := pkg.AllAssets()
			names := make([]string, 0, len(assets))
			for name := range assets {
				names = append(names, name)
			}
			slices.Sort(names)

			// stale entries first: a renamed asset must not see its old folder as foreign
			for name, entry := range lock {
				if _, ok := assets[name]; ok {
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

			for _, name := range names {
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

				installed, err := installAsset(cmd.Context(), lib, dir, a, entry)
				if err != nil {
					return err
				}
				// save after every asset so a later failure keeps what is installed
				lock[name] = installed
				if err := lock.Save(dir); err != nil {
					return err
				}
				cmd.Printf("installed %s (%s) -> %s\n", name, installed.VersionString, strings.Join(installed.Paths, ", "))
			}

			return nil
		},
	}
}
