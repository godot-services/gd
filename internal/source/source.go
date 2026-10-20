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
	"strconv"
	"strings"

	"github.com/godot-services/gd/internal/godot"
)

const DefaultLibraryURL = "https://godotengine.org/asset-library/api"

var (
	ErrNoDownloadURL      = errors.New("asset has no download url")
	ErrInsecureURL        = errors.New("git URL must use https")
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
	var asset struct {
		Title         string `json:"title"`
		VersionString string `json:"version_string"`
		DownloadURL   string `json:"download_url"`
	}
	if err := l.getJSON(ctx, "/asset/"+url.PathEscape(id), &asset); err != nil {
		return Resolved{}, fmt.Errorf("asset library: asset %s: %w", id, err)
	}
	if asset.DownloadURL == "" {
		return Resolved{}, fmt.Errorf("asset library: asset %s: %w", id, ErrNoDownloadURL)
	}

	return Resolved{URL: asset.DownloadURL, Title: asset.Title, VersionString: asset.VersionString, Source: ID(godot.PackageAssetConfiguration{Library: id})}, nil
}

type SearchResult struct {
	AssetID       string `json:"asset_id"`
	Title         string `json:"title"`
	VersionString string `json:"version_string"`
	GodotVersion  string `json:"godot_version"`
	Cost          string `json:"cost"`
	SupportLevel  string `json:"support_level"`
}

// Search lists addons matching query for a Godot version and returns them with the total match count.
func (l Library) Search(ctx context.Context, query, godotVersion string, max int) ([]SearchResult, int, error) {
	params := url.Values{
		"type":          {"addon"},
		"filter":        {query},
		"godot_version": {godotVersion},
		"max_results":   {strconv.Itoa(max)},
	}
	var page struct {
		Result     []SearchResult `json:"result"`
		TotalItems int            `json:"total_items"`
	}
	if err := l.getJSON(ctx, "/asset?"+params.Encode(), &page); err != nil {
		return nil, 0, fmt.Errorf("asset library: search: %w", err)
	}
	return page.Result, page.TotalItems, nil
}

func (l Library) getJSON(ctx context.Context, pathAndQuery string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(l.BaseURL, "/")+pathAndQuery, nil)
	if err != nil {
		return err
	}

	resp, err := l.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// GitArchiveURL builds the zip archive URL like the official asset library does.
func GitArchiveURL(repo, ref string) (string, error) {
	repo = trimRepo(repo)

	u, err := url.Parse(repo)
	if err != nil {
		return "", err
	}

	if u.Scheme != "https" {
		return "", fmt.Errorf("%w: %s", ErrInsecureURL, repo)
	}

	// escape per segment: refs like feature/x keep their slash, # or ? cannot cut the URL
	segments := strings.Split(ref, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	ref = strings.Join(segments, "/")

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

	return Resolved{URL: archive, Title: RepoName(a.Git), VersionString: a.Version, Source: ID(a)}, nil
}

// ID returns the lock source identifier of an asset, e.g. library:1234 or git:<repo>@<ref>.
func ID(a godot.PackageAssetConfiguration) string {
	if a.Library != "" {
		return "library:" + a.Library
	}
	return "git:" + trimRepo(a.Git) + "@" + a.Version
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
