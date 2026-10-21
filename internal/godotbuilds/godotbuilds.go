// Package godotbuilds reads the release layout of github.com/godotengine/godot-builds.
package godotbuilds

import (
	"bufio"
	"context"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/godot-services/gd/internal/install"
)

const (
	DefaultBaseURL = "https://github.com/godotengine/godot-builds/releases/download"
	DefaultAPIURL  = "https://api.github.com/repos/godotengine/godot-builds/releases"
)

var (
	ErrUnknownVersion = errors.New("unknown Godot version")
	ErrNoAsset        = errors.New("no editor asset for platform")
	ErrNoChecksum     = errors.New("no checksum")
	ErrNotModified    = errors.New("release list not modified")
)

// Client talks to godot-builds. Token is only for the API (APIURL), not for downloads.
type Client struct {
	BaseURL, APIURL, Token string
	HTTP                   *http.Client
}

// Sums returns asset name -> SHA-512 from the SHA512-SUMS.txt of tag.
func (c Client) Sums(ctx context.Context, tag string) (map[string]string, error) {
	url := c.BaseURL + "/" + tag + "/SHA512-SUMS.txt"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, fmt.Errorf("%w %s", ErrUnknownVersion, tag)
	default:
		return nil, fmt.Errorf("fetch %s: %s", url, resp.Status)
	}

	sums := map[string]string{}
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if sum, name, ok := strings.Cut(strings.TrimSpace(sc.Text()), "  "); ok {
			sums[strings.TrimSpace(name)] = sum
		}
	}
	return sums, sc.Err()
}

// SelectEditorAsset picks the editor zip for the platform from sums.
func SelectEditorAsset(sums map[string]string, goos, goarch string, dotnet bool) (string, error) {
	var suffixes []string
	switch goos {
	case "linux":
		if a, ok := map[string]string{"amd64": "x86_64", "arm64": "arm64"}[goarch]; ok {
			suffixes = []string{"_linux." + a + ".zip"}
			if dotnet {
				suffixes = []string{"_linux_" + a + ".zip"}
			}
		}
	case "darwin":
		if a, ok := map[string]string{"amd64": "x86_64", "arm64": "arm64"}[goarch]; ok {
			suffixes = []string{"_macos.universal.zip", "_macos." + a + ".zip"}
		}
	case "windows":
		switch goarch {
		case "amd64":
			suffixes = []string{"_win64.exe.zip"}
			if dotnet {
				suffixes = []string{"_win64.zip"}
			}
		case "arm64":
			suffixes = []string{"_windows_arm64.exe.zip"}
			if dotnet {
				suffixes = []string{"_windows_arm64.zip"}
			}
		}
	}

	var editors []string
	for name := range sums {
		if strings.HasPrefix(name, "Godot_v") && strings.HasSuffix(name, ".zip") &&
			!strings.Contains(name, "debugsymbols") && !strings.Contains(name, "dSYM") {
			editors = append(editors, name)
		}
	}
	slices.Sort(editors)

	for _, suffix := range suffixes {
		for _, name := range editors {
			// "_mono_macos.arm64.zip" also ends with the standard suffix
			if strings.HasSuffix(name, suffix) && strings.Contains(name, "_mono_") == dotnet {
				return name, nil
			}
		}
	}
	return "", fmt.Errorf("%w %s/%s: found %s", ErrNoAsset, goos, goarch, strings.Join(editors, ", "))
}

// TemplatesAsset is the export templates asset name of tag.
func TemplatesAsset(tag string, dotnet bool) string {
	if dotnet {
		return "Godot_v" + tag + "_mono_export_templates.tpz"
	}
	return "Godot_v" + tag + "_export_templates.tpz"
}

// Download fetches asset of tag into dir and verifies it against wantSHA512.
func (c Client) Download(ctx context.Context, tag, asset, wantSHA512, dir string) (string, error) {
	if wantSHA512 == "" {
		return "", fmt.Errorf("%w for %s", ErrNoChecksum, asset)
	}
	path, _, err := install.FetchHash(ctx, c.HTTP, c.BaseURL+"/"+tag+"/"+asset, wantSHA512, dir, sha512.New)
	return path, err
}

var godot4TagRe = regexp.MustCompile(`^4\.\d+(\.\d+)?(-(stable|rc\d+|beta\d+|dev\d+))?$`)

// Releases lists Godot 4 release tags, newest first (API order), and the ETag of
// the first page. With etag set and an unchanged first page it returns
// ErrNotModified: new releases always land on the first page.
func (c Client) Releases(ctx context.Context, etag string) (tags []string, newETag string, err error) {
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s?per_page=100&page=%d", c.APIURL, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
		if page == 1 && etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		releases, pageETag, err := c.releasePage(req)
		if err != nil {
			return nil, "", err
		}
		if page == 1 {
			newETag = pageETag
		}
		if len(releases) == 0 {
			return tags, newETag, nil
		}
		for _, r := range releases {
			if godot4TagRe.MatchString(r.TagName) {
				tags = append(tags, r.TagName)
			}
		}
	}
}

type release struct {
	TagName string `json:"tag_name"`
}

func (c Client) releasePage(req *http.Request) ([]release, string, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return nil, "", ErrNotModified
	}
	if resp.StatusCode != http.StatusOK {
		hint := ""
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			hint = " (set GH_TOKEN to raise the rate limit)"
		}
		return nil, "", fmt.Errorf("fetch %s: %s%s", req.URL, resp.Status, hint)
	}
	var releases []release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, "", fmt.Errorf("decode %s: %w", req.URL, err)
	}
	return releases, resp.Header.Get("ETag"), nil
}
