package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/godot-services/gd/internal/godotbuilds"
	"github.com/godot-services/gd/internal/source"
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	opts := &options{}
	info, ok := debug.ReadBuildInfo()

	rootCmd := &cobra.Command{
		Use:   "gd",
		Short: "Package manager for Godot projects",
		Long: `gd manages the addons of a Godot project.
Assets are declared in package.godot and pinned in package-lock.godot.`,
		Version:       resolveVersion(buildVersion, info, ok),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&opts.libraryURL, "library-url", source.DefaultLibraryURL, "asset library API base URL")
	rootCmd.PersistentFlags().MarkHidden("library-url")

	rootCmd.PersistentFlags().StringVar(&opts.buildsURL, "builds-url", godotbuilds.DefaultBaseURL, "godot-builds download base URL")
	rootCmd.PersistentFlags().StringVar(&opts.buildsAPIURL, "builds-api-url", godotbuilds.DefaultAPIURL, "godot-builds releases API URL")
	rootCmd.PersistentFlags().MarkHidden("builds-url")
	rootCmd.PersistentFlags().MarkHidden("builds-api-url")

	rootCmd.AddCommand(newEditorCmd(opts), newAddCmd(opts), newInstallCmd(opts), newRemoveCmd(), newSearchCmd(opts), newOutdatedCmd(opts), newUpdateCmd(opts), newCacheCmd(), newOpenCmd("open"), newGodotCmd(), newStartCmd(), newTemplatesCmd(opts), newConfigCmd())

	return rootCmd
}

// exitCode reports err on stderr (exitError stays silent: Godot already spoke) and returns the process exit code.
func exitCode(err error, stderr io.Writer) int {
	var ee exitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ee):
		return ee.code
	}
	fmt.Fprintln(stderr, "Error:", err)
	return 1
}

// Execute runs the root command. This is called by main.main().
func Execute() {
	rootCmd := newRootCmd()
	if code := exitCode(rootCmd.Execute(), rootCmd.ErrOrStderr()); code != 0 {
		os.Exit(code)
	}
}
