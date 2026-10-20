// Package source resolves an asset reference to the zip that holds it.
package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/godot-services/gd/internal/godot"
)

const DefaultLibraryURL = "https://godotengine.org/asset-library/api"

var (
	ErrNoDownloadURL      = errors.New("asset has no download url")
	ErrUnsupportedGitHost = errors.New("unsupported git host (supported: github.com, gitlab.com, codeberg.org)")

	nonSlugRe = regexp.MustCompile(`[^a-z0-9_]+`)
)

type Resolved struct {
	URL           string
	Title         string
	VersionString string
	Source        string
}

type Library struct {
	BaseURL string
	Client  *http.Client
}

// Resolve looks up an asset in the asset library (GET {BaseURL}/asset/{id}).
func (l Library) Resolve(ctx context.Context, id string) (Resolved, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(l.BaseURL, "/")+"/asset/"+url.PathEscape(id), nil)
	if err != nil {
		return Resolved{}, err
	}

	client := l.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Resolved{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Resolved{}, fmt.Errorf("asset library: asset %s: %s", id, resp.Status)
	}

	var asset struct {
		Title         string `json:"title"`
		VersionString string `json:"version_string"`
		DownloadURL   string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&asset); err != nil {
		return Resolved{}, fmt.Errorf("asset library: asset %s: %w", id, err)
	}
	if asset.DownloadURL == "" {
		return Resolved{}, fmt.Errorf("asset library: asset %s: %w", id, ErrNoDownloadURL)
	}

	return Resolved{URL: asset.DownloadURL, Title: asset.Title, VersionString: asset.VersionString, Source: "library:" + id}, nil
}

// GitArchiveURL builds the zip archive URL like the official asset library does.
func GitArchiveURL(repo, ref string) (string, error) {
	repo = trimRepo(repo)

	u, err := url.Parse(repo)
	if err != nil {
		return "", err
	}

	switch u.Host {
	case "github.com", "codeberg.org":
		return repo + "/archive/" + ref + ".zip", nil
	case "gitlab.com":
		return repo + "/-/archive/" + ref + ".zip", nil
	}

	return "", fmt.Errorf("%w: %s", ErrUnsupportedGitHost, u.Host)
}

func Resolve(ctx context.Context, lib Library, a godot.PackageAssetConfiguration) (Resolved, error) {
	if a.Library != "" {
		return lib.Resolve(ctx, a.Library)
	}

	archive, err := GitArchiveURL(a.Git, a.Version)
	if err != nil {
		return Resolved{}, err
	}

	return Resolved{URL: archive, Title: RepoName(a.Git), VersionString: a.Version, Source: "git:" + trimRepo(a.Git) + "@" + a.Version}, nil
}

// Slug turns a title into an asset name: lowercase, [a-z0-9_] only.
func Slug(s string) string {
	return strings.Trim(nonSlugRe.ReplaceAllString(strings.ToLower(s), "_"), "_")
}

// RepoName returns the last path segment of a repository URL without .git.
func RepoName(repo string) string {
	return path.Base(trimRepo(repo))
}

func trimRepo(repo string) string {
	return strings.TrimSuffix(strings.TrimSuffix(repo, "/"), ".git")
}
