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
			dir, zips, size, err := cachedZips()
			if err != nil {
				return err
			}
			cmd.Printf("%s\n%s, %.1f MB\n", dir, plural(len(zips), "file"), float64(size)/1e6)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "clean",
		Short: "Delete all cached downloads",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, zips, size, err := cachedZips()
			if err != nil {
				return err
			}
			for _, zip := range zips {
				if err := os.Remove(zip); err != nil {
					return err
				}
			}
			cmd.Printf("removed %s, %.1f MB\n", plural(len(zips), "file"), float64(size)/1e6)
			return nil
		},
	})

	return cmd
}

// cachedZips lists the *.zip files in the cache dir; only these are ever deleted.
func cachedZips() (dir string, zips []string, size int64, err error) {
	if dir, err = cacheDir(); err != nil {
		return "", nil, 0, err
	}
	if zips, err = filepath.Glob(filepath.Join(dir, "*.zip")); err != nil {
		return "", nil, 0, err
	}
	for _, zip := range zips {
		info, err := os.Stat(zip)
		if err != nil {
			return "", nil, 0, err
		}
		size += info.Size()
	}
	return dir, zips, size, nil
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
