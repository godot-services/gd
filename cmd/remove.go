package cmd

import (
	"github.com/godot-services/gd/internal/install"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an asset and its installed addons",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			dir, pkg, lock, err := loadProject()
			if err != nil {
				return err
			}
			if err := pkg.RemoveAsset(name); err != nil {
				return err
			}
			if err := install.Remove(dir, lock[name].Paths); err != nil {
				return err
			}
			if err := pkg.Save(dir); err != nil {
				return err
			}
			delete(lock, name)
			if err := lock.Save(dir); err != nil {
				return err
			}

			cmd.Printf("removed %s\n", name)
			return nil
		},
	}
}
