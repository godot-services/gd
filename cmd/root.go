package cmd

import (
	"errors"
	"os"

	"github.com/godot-services/gd/internal/godotbuilds"
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

	rootCmd.PersistentFlags().StringVar(&opts.buildsURL, "builds-url", godotbuilds.DefaultBaseURL, "godot-builds download base URL")
	rootCmd.PersistentFlags().StringVar(&opts.buildsAPIURL, "builds-api-url", godotbuilds.DefaultAPIURL, "godot-builds releases API URL")
	rootCmd.PersistentFlags().MarkHidden("builds-url")
	rootCmd.PersistentFlags().MarkHidden("builds-api-url")

	rootCmd.AddCommand(newEditorCmd(opts), newAddCmd(opts), newInstallCmd(opts), newRemoveCmd(), newSearchCmd(opts), newOutdatedCmd(opts), newUpdateCmd(opts), newCacheCmd(), newOpenCmd("open"), newGodotCmd())

	return rootCmd
}

// Execute runs the root command. This is called by main.main().
func Execute() {
	err := newRootCmd().Execute()
	var ee exitError
	if errors.As(err, &ee) {
		os.Exit(ee.code)
	}
	if err != nil {
		os.Exit(1)
	}
}
