package cmd

import (
	"errors"
	"runtime"

	"github.com/godot-services/gd/internal/editor"
	"github.com/godot-services/gd/internal/godot"
	"github.com/spf13/cobra"
)

func newGodotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "godot -- [args...]",
		Short: "Run the project's Godot editor with the given arguments",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, e, err := resolveProjectEditor(cmd)
			if errors.Is(err, godot.ErrNoGodotProject) {
				// outside a project the default alias applies; open/start still need one
				var reg *editor.Registry
				if reg, _, err = loadRegistry(); err == nil {
					e, err = reg.Resolve(editor.Requirement{})
				}
			}
			if err != nil {
				return err
			}
			path := e.Path
			if runtime.GOOS == "windows" {
				path = editor.ConsolePath(path)
			}
			return runEditorProcess(cmd, path, args)
		},
	}
}
