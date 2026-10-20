package cmd

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"text/tabwriter"

	"github.com/godot-services/gd/internal/editor"
	"github.com/godot-services/gd/internal/godot"
	"github.com/spf13/cobra"
)

func newEditorCmd(opts *options) *cobra.Command {
	c := &cobra.Command{Use: "editor", Short: "Manage Godot editors"}
	c.AddCommand(newEditorInstallCmd(opts), newEditorUninstallCmd(), newEditorListCmd(), newEditorAvailableCmd(opts))
	return c
}

func newEditorInstallCmd(opts *options) *cobra.Command {
	var dotnet bool
	c := &cobra.Command{
		Use:   "install [version]",
		Short: "Download and register an official Godot editor",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var version string
			if len(args) == 1 {
				version = args[0]
			} else {
				_, pkg, _, err := loadProject()
				if err != nil || pkg.Editor.Version == "" {
					return errors.New("no [editor] version in package.godot, pass a version: gd editor install <version>")
				}
				version = pkg.Editor.Version
				if !cmd.Flags().Changed("dotnet") {
					dotnet = pkg.Editor.Dotnet
				}
			}
			dataDir, err := editor.DataDir()
			if err != nil {
				return err
			}
			e, err := editor.Install(cmd.Context(), opts.builds(), dataDir, version, dotnet, runtime.GOOS, runtime.GOARCH)
			if errors.Is(err, editor.ErrAlreadyInstalled) {
				cmd.Println(err)
				return nil
			}
			if err != nil {
				return err
			}
			reg, err := editor.LoadRegistry(dataDir)
			if err != nil {
				return err
			}
			reg.Editors[e.Name] = e
			if reg.Default == "" {
				reg.Default = e.Name
			}
			if err := reg.Save(dataDir); err != nil {
				return err
			}
			cmd.Printf("installed %s (%s) -> %s\n", e.Name, e.Version, e.Path)
			return nil
		},
	}
	c.Flags().BoolVar(&dotnet, "dotnet", false, "install the .NET (mono) flavour")
	return c
}

func newEditorUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <name>",
		Short: "Remove a managed editor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			dataDir, err := editor.DataDir()
			if err != nil {
				return err
			}
			reg, err := editor.LoadRegistry(dataDir)
			if err != nil {
				return err
			}
			e, ok := reg.Editors[name]
			if !ok {
				return fmt.Errorf("%w %q", editor.ErrUnknownEditor, name)
			}
			if e.Source != "managed" {
				return fmt.Errorf("editor %q is linked, use gd editor unlink %s", name, name)
			}
			// the entry's Path is never trusted for deletion, only the managed dir
			if err := os.RemoveAll(filepath.Join(dataDir, "editors", name)); err != nil {
				return err
			}
			delete(reg.Editors, name)
			if reg.Default == name {
				reg.Default = ""
			}
			if err := reg.Save(dataDir); err != nil {
				return err
			}
			cmd.Printf("uninstalled %s\n", name)
			return nil
		},
	}
}

func newEditorListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered editors",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dataDir, err := editor.DataDir()
			if err != nil {
				return err
			}
			reg, err := editor.LoadRegistry(dataDir)
			if err != nil {
				return err
			}
			if len(reg.Editors) == 0 {
				cmd.Println("no editors installed, run gd editor install <version>")
				return nil
			}
			var resolved string
			if dir, pkg, _, err := loadProject(); err == nil {
				local, _ := godot.LoadLocalEditor(dir)
				if local != "" || pkg.Editor.Version != "" {
					req := editor.Requirement{Local: local, Dotnet: pkg.Editor.Dotnet}
					if pkg.Editor.Version != "" {
						req.Version = godot.ReleaseTag(pkg.Editor.Version)
					}
					if e, err := reg.Resolve(req); err == nil {
						resolved = e.Name
					}
				}
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, " \tNAME\tVERSION\tDOTNET\tBUILD\tSOURCE\tPATH")
			for _, n := range slices.Sorted(maps.Keys(reg.Editors)) {
				e, mark := reg.Editors[n], " "
				if n == resolved {
					mark = ">"
				} else if n == reg.Default {
					mark = "*"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%s\t%s\t%s\n", mark, e.Name, e.Version, e.Dotnet, e.Build, e.Source, e.Path)
			}
			return w.Flush()
		},
	}
}

func newEditorAvailableCmd(opts *options) *cobra.Command {
	var pre bool
	c := &cobra.Command{
		Use:   "available",
		Short: "List Godot 4 versions from godot-builds",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, err := opts.builds().Available(cmd.Context(), pre)
			if err != nil {
				return err
			}
			if len(tags) == 0 {
				cmd.Println("no Godot 4 releases found")
			}
			for _, t := range tags {
				cmd.Println(t)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&pre, "pre", false, "include rc, beta and dev releases")
	return c
}
