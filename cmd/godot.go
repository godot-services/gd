package cmd

import (
	"runtime"

	"github.com/godot-services/gd/internal/editor"
	"github.com/spf13/cobra"
)

func newGodotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "godot -- [args...]",
		Short: "Run the project's Godot editor with the given arguments",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, e, err := resolveEditor(cmd)
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
