package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/godot-services/gd/internal/editor"
	"github.com/godot-services/gd/internal/godot"
	"github.com/spf13/cobra"
)

// sceneArg turns a scene argument into a Godot scene reference.
func sceneArg(dir, scene string) (string, error) {
	if strings.HasPrefix(scene, "res://") || strings.HasPrefix(scene, "uid://") {
		return scene, nil
	}
	rel := path.Clean(filepath.ToSlash(scene))
	if rel == ".." || strings.HasPrefix(rel, "../") || path.IsAbs(rel) || filepath.IsAbs(scene) {
		return "", fmt.Errorf("scene %s is outside the project", scene)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
		return "", fmt.Errorf("scene %s: %w", scene, err)
	}
	return "res://" + rel, nil
}

func newStartCmd() *cobra.Command {
	var detach bool
	c := &cobra.Command{
		Use:   "start [scene]",
		Short: "Run the game with the project's Godot editor",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, e, err := resolveProjectEditor(cmd)
			if err != nil {
				return err
			}
			a := []string{"--path", dir}
			if len(args) == 1 {
				scene, err := sceneArg(dir, args[0])
				if err != nil {
					return err
				}
				a = append(a, scene)
			} else if _, err := (&godot.Project{Location: dir}).MainScene(); errors.Is(err, godot.ErrNoMainScene) {
				return fmt.Errorf("%w: set run/main_scene in project.godot or pass a scene", err)
			} else if err != nil {
				return err
			}
			if detach {
				return editor.StartDetached(e.Path, a)
			}
			return runEditorProcess(cmd, e.Path, a)
		},
	}
	c.Flags().BoolVar(&detach, "detach", false, "start the game and return immediately")
	return c
}
