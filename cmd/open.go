package cmd

import (
	"errors"
	"fmt"

	"github.com/godot-services/gd/internal/editor"
	"github.com/godot-services/gd/internal/godot"
	"github.com/spf13/cobra"
)

// exitError carries the exit code of a foreground editor run to Execute.
type exitError struct{ code int }

func (e exitError) Error() string { return fmt.Sprintf("godot exited with status %d", e.code) }

// resolveProjectEditor finds the editor for the project in the working directory,
// printing version and local-editor mismatch warnings to stderr.
func resolveProjectEditor(cmd *cobra.Command) (dir string, e editor.Editor, err error) {
	project, err := godot.NewProject(".")
	if err != nil {
		return "", e, err
	}
	dir = project.Location
	pkg, err := godot.LoadPackage(dir)
	if err != nil {
		return "", e, err
	}
	local, err := godot.LoadLocalEditor(dir)
	if err != nil {
		return "", e, err
	}
	dataDir, err := editor.DataDir()
	if err != nil {
		return "", e, err
	}
	reg, err := editor.LoadRegistry(dataDir)
	if err != nil {
		return "", e, err
	}
	req := editor.Requirement{Dotnet: pkg.Editor.Dotnet, Local: local}
	if pkg.Editor.Version != "" {
		req.Version = godot.ReleaseTag(pkg.Editor.Version)
	}
	if e, err = reg.Resolve(req); err != nil {
		return "", e, err
	}
	if v, verr := project.GodotVersion(); verr == nil {
		if w := editor.VersionWarning(v, e.Version); w != "" {
			cmd.PrintErrln(w)
		}
	}
	if local != "" && req.Version != "" && (e.Version != req.Version || e.Dotnet != req.Dotnet) {
		cmd.PrintErrf("warning: editor %s (%s) differs from [editor] in package.godot (%s)\n", e.Name, e.Version, req.Version)
	}
	return dir, e, nil
}

// resolveEditor is resolveProjectEditor, but outside a project it resolves the
// default editor and returns an empty dir.
func resolveEditor(cmd *cobra.Command) (dir string, e editor.Editor, err error) {
	dir, e, err = resolveProjectEditor(cmd)
	if errors.Is(err, godot.ErrNoGodotProject) {
		var reg *editor.Registry
		if reg, _, err = loadRegistry(); err == nil {
			e, err = reg.Resolve(editor.Requirement{})
		}
	}
	return dir, e, err
}

func runEditorProcess(cmd *cobra.Command, path string, args []string) error {
	code, err := editor.Run(cmd.Context(), path, args, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	if code != 0 {
		return exitError{code}
	}
	return nil
}

func newOpenCmd(use string) *cobra.Command {
	var detach bool
	c := &cobra.Command{
		Use:   use,
		Short: "Open the project in its Godot editor (outside a project: the project manager)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, e, err := resolveEditor(cmd)
			if err != nil {
				return err
			}
			a := []string{"-e", "--path", dir}
			if dir == "" {
				a = []string{"--project-manager"}
			}
			if detach {
				return editor.StartDetached(e.Path, a)
			}
			return runEditorProcess(cmd, e.Path, a)
		},
	}
	c.Flags().BoolVar(&detach, "detach", false, "start the editor and return immediately")
	return c
}
