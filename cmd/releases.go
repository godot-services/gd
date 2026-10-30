package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/godot-services/gd/internal/godotbuilds"
	"github.com/spf13/cobra"
)

const (
	releasesCacheFile   = "godot-builds-releases.json"
	releasesCacheMaxAge = time.Hour
)

type releaseCache struct {
	ETag    string    `json:"etag"`
	Fetched time.Time `json:"fetched"`
	Tags    []string  `json:"tags"`
}

// releases lists all Godot 4 tags of godot-builds, cached in the cache dir (ADR 0025).
// A fresh cache needs no request, an older one is revalidated with its ETag, and
// a failed request falls back to the cache with a warning.
func releases(cmd *cobra.Command, c godotbuilds.Client, refresh bool) ([]string, error) {
	dir, err := cacheDir()
	if err != nil {
		tags, _, err := c.Releases(cmd.Context(), "")
		return tags, err
	}
	path := filepath.Join(dir, releasesCacheFile)

	var cached releaseCache
	b, err := os.ReadFile(path)
	ok := err == nil && json.Unmarshal(b, &cached) == nil && !cached.Fetched.IsZero()
	if !ok {
		cached = releaseCache{}
	}
	// a negative age (clock moved back) counts as stale, not as fresh forever
	if age := time.Since(cached.Fetched); ok && !refresh && age >= 0 && age < releasesCacheMaxAge {
		return cached.Tags, nil
	}

	tags, etag, err := c.Releases(cmd.Context(), cached.ETag)
	switch {
	case errors.Is(err, godotbuilds.ErrNotModified):
		tags, etag = cached.Tags, cached.ETag
	case err != nil && ok && cmd.Context().Err() == nil:
		cmd.PrintErrf("warning: using cached release list from %s: %v\n", cached.Fetched.Local().Format(time.DateTime), err)
		return cached.Tags, nil
	case err != nil:
		return nil, err
	}

	// the cache only saves requests; failing to write it must not fail the command
	if b, err := json.Marshal(releaseCache{ETag: etag, Fetched: time.Now(), Tags: tags}); err == nil {
		writeAtomic(path, b)
	}
	return tags, nil
}

// writeAtomic replaces path via a temp file, so concurrent readers never see a partial file.
func writeAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}
