package cmd

import (
	"os"

	"github.com/godot-services/gd/internal/source"
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	opts := &options{}

	rootCmd := &cobra.Command{
		Use:   "gd",
		Short: "Package manager for Godot projects",
		Long: `gd manages the addons of a Godot project.
Assets are declared in package.godot and pinned in package-lock.godot.`,
		SilenceUsage: true,
	}

	rootCmd.PersistentFlags().StringVar(&opts.libraryURL, "library-url", source.DefaultLibraryURL, "asset library API base URL")
	rootCmd.PersistentFlags().MarkHidden("library-url")

	rootCmd.AddCommand(newAddCmd(opts), newInstallCmd(opts), newRemoveCmd())

	return rootCmd
}

// Execute runs the root command. This is called by main.main().
func Execute() {
	err := newRootCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}
