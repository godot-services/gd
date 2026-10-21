package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/godot-services/gd/internal/godot"
	"github.com/spf13/cobra"
)

func newSearchCmd(opts *options) *cobra.Command {
	var godotVersion string
	var max int

	cmd := &cobra.Command{
		Use:   "search <text>",
		Short: "Search addons in the asset library",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if max < 1 || max > 500 {
				return fmt.Errorf("--max must be between 1 and 500, got %d", max)
			}

			if godotVersion == "" {
				project, err := godot.NewProject(".")
				if err != nil {
					return fmt.Errorf("%w (or pass --godot)", err)
				}
				pkg, err := godot.LoadPackage(project.Location)
				if err != nil {
					return err
				}
				if pkg.Editor.Version != "" {
					godotVersion = godot.MajorMinor(pkg.Editor.Version)
					warnVersionMismatch(cmd, project, pkg)
				} else if godotVersion, err = project.GodotVersion(); err != nil {
					return err
				}
			}

			query := strings.Join(args, " ")
			results, total, err := opts.library().Search(cmd.Context(), query, godotVersion, max)
			if err != nil {
				return err
			}
			if len(results) == 0 {
				cmd.Printf("no assets found for %q (Godot %s)\n", query, godotVersion)
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tVERSION\tGODOT\tLICENSE\tSUPPORT")
			for _, r := range results {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", r.AssetID, r.Title, r.VersionString, r.GodotVersion, r.Cost, r.SupportLevel)
			}
			if err := w.Flush(); err != nil {
				return err
			}

			cmd.Printf("%d of %d results. Install with: gd add <ID>\n", len(results), total)
			return nil
		},
	}

	cmd.Flags().StringVar(&godotVersion, "godot", "", "Godot version to search for (default: [editor] version of package.godot, else project.godot)")
	cmd.Flags().IntVar(&max, "max", 20, "maximum number of results (1-500)")

	return cmd
}
