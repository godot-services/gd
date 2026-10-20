package cmd

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/godot-services/gd/internal/editor"
	"github.com/godot-services/gd/internal/godot"
	"github.com/spf13/cobra"
)

func newTemplatesCmd(opts *options) *cobra.Command {
	c := &cobra.Command{Use: "templates", Short: "Manage Godot export templates"}
	c.AddCommand(newTemplatesInstallCmd(opts), newTemplatesListCmd(), newTemplatesUninstallCmd())
	return c
}

// templatesDirFor is the self-contained templates dir of the project's editor, else Godot's own.
func templatesDirFor(cmd *cobra.Command, inProject bool) (string, error) {
	if inProject {
		// resolution problems must not block installing templates
		if _, e, err := resolveProjectEditor(cmd); err == nil {
			if d, ok := editor.SelfContainedTemplatesDir(e.Path); ok {
				return d, nil
			}
		}
	}
	return godotTemplatesDir()
}

func godotTemplatesDir() (string, error) {
	d, err := editor.GodotDataDir()
	if err != nil {
		return "", err
	}
	return editor.TemplatesDir(d), nil
}

func newTemplatesInstallCmd(opts *options) *cobra.Command {
	var dotnet, force bool
	c := &cobra.Command{
		Use:   "install [version]",
		Short: "Download and install export templates",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if err := godot.ValidateVersion(args[0]); err != nil {
					return err
				}
			}
			_, pkg, _, perr := loadProject()
			var version string
			if len(args) == 1 {
				version = args[0]
			} else {
				if perr != nil || pkg.Editor.Version == "" {
					return errors.New("no [editor] version in package.godot, pass a version: gd templates install <version>")
				}
				version = pkg.Editor.Version
			}
			if perr == nil && len(args) == 0 && !cmd.Flags().Changed("dotnet") {
				dotnet = pkg.Editor.Dotnet
			}
			dir, err := templatesDirFor(cmd, perr == nil)
			if err != nil {
				return err
			}
			path, err := editor.InstallTemplates(cmd.Context(), opts.builds(), godot.ReleaseTag(version), dotnet, dir, force)
			if errors.Is(err, editor.ErrTemplatesInstalled) {
				_, name, _ := strings.Cut(err.Error(), ": ")
				cmd.Printf("templates %s already installed (use --force)\n", name)
				return nil
			}
			if err != nil {
				return err
			}
			cmd.Printf("installed templates %s -> %s\n", filepath.Base(path), path)
			return nil
		},
	}
	c.Flags().BoolVar(&dotnet, "dotnet", false, "install the .NET (mono) flavour")
	c.Flags().BoolVar(&force, "force", false, "replace templates that are already installed")
	return c
}

func newTemplatesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed export templates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := godotTemplatesDir()
			if err != nil {
				return err
			}
			names, err := editor.ListTemplates(dir)
			if err != nil {
				return err
			}
			if len(names) == 0 {
				cmd.Println("no export templates installed")
			}
			for _, n := range names {
				cmd.Println(n)
			}
			return nil
		},
	}
}

func newTemplatesUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <version>",
		Short: "Remove installed export templates",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := godotTemplatesDir()
			if err != nil {
				return err
			}
			if err := editor.UninstallTemplates(dir, args[0]); err != nil {
				return err
			}
			cmd.Printf("uninstalled templates %s\n", args[0])
			return nil
		},
	}
}
