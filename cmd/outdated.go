package cmd

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"text/tabwriter"

	"github.com/godot-services/gd/internal/source"
	"github.com/spf13/cobra"
)

// errOutdated makes gd outdated exit with 1, like npm outdated, so CI can react.
var errOutdated = errors.New("assets are outdated")

func newOutdatedCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "outdated",
		Short: "Show library assets that have a newer version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, pkg, lock, err := loadProject()
			if err != nil {
				return err
			}

			lib := opts.library()
			assets := pkg.AllAssets()
			var rows [][4]string
			outdated := 0

			for _, name := range slices.Sorted(maps.Keys(assets)) {
				a := assets[name]
				entry, locked := lock[name]
				switch {
				case !locked:
					rows = append(rows, [4]string{name, "-", "not installed", source.ID(a)})
				case a.Git != "":
					rows = append(rows, [4]string{name, a.Version, "pinned", source.ID(a)})
				default:
					latest, err := source.Resolve(cmd.Context(), lib, a)
					if err != nil {
						return err
					}
					if latest.URL != entry.URL || latest.VersionString != entry.VersionString {
						rows = append(rows, [4]string{name, entry.VersionString, latest.VersionString, source.ID(a)})
						outdated++
					}
				}
			}

			if len(rows) == 0 {
				cmd.Println("all assets are up to date")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tCURRENT\tLATEST\tSOURCE")
			for _, r := range rows {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r[0], r[1], r[2], r[3])
			}
			if err := w.Flush(); err != nil {
				return err
			}

			if outdated > 0 {
				return fmt.Errorf("%d %w", outdated, errOutdated)
			}
			return nil
		},
	}
}
