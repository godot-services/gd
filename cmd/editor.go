package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/godot-services/gd/internal/editor"
	"github.com/godot-services/gd/internal/godot"
	"github.com/spf13/cobra"
)

func newEditorCmd(opts *options) *cobra.Command {
	c := &cobra.Command{Use: "editor", Short: "Manage Godot editors"}
	c.AddCommand(newEditorInstallCmd(opts), newEditorUninstallCmd(), newEditorListCmd(), newEditorAvailableCmd(opts),
		newEditorLinkCmd(), newEditorUnlinkCmd(), newEditorDefaultCmd(), newEditorUseCmd(), newOpenCmd("open"))
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
				if err := godot.ValidateVersion(args[0]); err != nil {
					return err
				}
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

var aliasRe = regexp.MustCompile(`^[a-z0-9_-]+$`)

func loadRegistry() (*editor.Registry, string, error) {
	dataDir, err := editor.DataDir()
	if err != nil {
		return nil, "", err
	}
	reg, err := editor.LoadRegistry(dataDir)
	return reg, dataDir, err
}

func newEditorLinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "link <alias> <path>",
		Short: "Register a Godot editor installed elsewhere",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			if !aliasRe.MatchString(alias) {
				return fmt.Errorf("invalid alias %q: use lowercase letters, digits, - and _", alias)
			}
			if alias == "aliases" {
				return editor.ErrReservedName
			}
			path, err := filepath.Abs(args[1])
			if err != nil {
				return err
			}
			fi, err := os.Stat(path)
			if err != nil {
				return err
			}
			if runtime.GOOS == "darwin" && fi.IsDir() && strings.HasSuffix(path, ".app") {
				path = filepath.Join(path, "Contents", "MacOS", "Godot")
			}
			reg, dataDir, err := loadRegistry()
			if err != nil {
				return err
			}
			if _, ok := reg.Editors[alias]; ok {
				return fmt.Errorf("editor %q already exists", alias)
			}
			version, dotnet, build, err := editor.ProbeVersion(cmd.Context(), path)
			if err != nil {
				return err
			}
			reg.Editors[alias] = editor.Editor{Name: alias, Path: path, Version: version, Dotnet: dotnet, Build: build, Source: "linked"}
			if err := reg.Save(dataDir); err != nil {
				return err
			}
			cmd.Printf("linked %s (%s, %s) -> %s\n", alias, version, build, path)
			return nil
		},
	}
}

func newEditorUnlinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlink <alias>",
		Short: "Forget a linked editor (its files are kept)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			reg, dataDir, err := loadRegistry()
			if err != nil {
				return err
			}
			e, ok := reg.Editors[name]
			if !ok {
				return fmt.Errorf("%w %q", editor.ErrUnknownEditor, name)
			}
			if e.Source != "linked" {
				return fmt.Errorf("editor %q is managed, use gd editor uninstall %s", name, name)
			}
			delete(reg.Editors, name)
			if reg.Default == name {
				reg.Default = ""
			}
			if err := reg.Save(dataDir); err != nil {
				return err
			}
			cmd.Printf("unlinked %s\n", name)
			return nil
		},
	}
}

func newEditorDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "default <name>",
		Short: "Set the default editor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, dataDir, err := loadRegistry()
			if err != nil {
				return err
			}
			if _, ok := reg.Editors[args[0]]; !ok {
				return fmt.Errorf("%w %q", editor.ErrUnknownEditor, args[0])
			}
			reg.Default = args[0]
			if err := reg.Save(dataDir); err != nil {
				return err
			}
			cmd.Printf("default editor: %s\n", args[0])
			return nil
		},
	}
}

func newEditorUseCmd() *cobra.Command {
	var clear bool
	c := &cobra.Command{
		Use:   "use [name]",
		Short: "Use a specific editor for this project on this machine",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if clear == (len(args) == 1) {
				return errors.New("pass an editor name or --clear")
			}
			project, err := godot.NewProject(".")
			if err != nil {
				return err
			}
			if clear {
				return godot.SaveLocalEditor(project.Location, "")
			}
			reg, _, err := loadRegistry()
			if err != nil {
				return err
			}
			if _, ok := reg.Editors[args[0]]; !ok {
				return fmt.Errorf("%w %q", editor.ErrUnknownEditor, args[0])
			}
			if err := godot.SaveLocalEditor(project.Location, args[0]); err != nil {
				return err
			}
			cmd.Printf("using editor %s for this project\n", args[0])
			if !gitignored(project.Location) {
				cmd.PrintErrln("hint: add " + godot.LocalFileName + " to .gitignore")
			}
			return nil
		},
	}
	c.Flags().BoolVar(&clear, "clear", false, "remove the per-machine editor choice")
	return c
}

// gitignored reports true when there is no .gitignore to extend or it already lists the local file.
func gitignored(dir string) bool {
	f, err := os.Open(filepath.Join(dir, ".gitignore"))
	if err != nil {
		return true
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if l := strings.TrimSpace(sc.Text()); l == godot.LocalFileName || l == "/"+godot.LocalFileName {
			return true
		}
	}
	return false
}
