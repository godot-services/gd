package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Inspect or clear the download cache",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "info",
		Short: "Show where the cache is and how big it is",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, files, size, err := cachedFiles()
			if err != nil {
				return err
			}
			cmd.Printf("%s\n%s, %.1f MB\n", dir, plural(len(files), "file"), float64(size)/1e6)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "clean",
		Short: "Delete all cached downloads",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, files, size, err := cachedFiles()
			if err != nil {
				return err
			}
			for _, f := range files {
				if err := os.Remove(f); err != nil {
					return err
				}
			}
			cmd.Printf("removed %s, %.1f MB\n", plural(len(files), "file"), float64(size)/1e6)
			return nil
		},
	})

	return cmd
}

// cachedFiles lists the *.zip files and the release list in the cache dir; only these are ever deleted.
func cachedFiles() (dir string, files []string, size int64, err error) {
	if dir, err = cacheDir(); err != nil {
		return "", nil, 0, err
	}
	if files, err = filepath.Glob(filepath.Join(dir, "*.zip")); err != nil {
		return "", nil, 0, err
	}
	releases := filepath.Join(dir, releasesCacheFile)
	if _, err := os.Stat(releases); err == nil {
		files = append(files, releases)
	}
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return "", nil, 0, err
		}
		size += info.Size()
	}
	return dir, files, size, nil
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
