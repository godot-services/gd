package editor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var ErrUnknownVersionOutput = errors.New("unrecognized `godot --version` output")

var versionRe = regexp.MustCompile(`^(\d+\.\d+(?:\.\d+)?)\.(stable|rc\d+|beta\d+|dev\d+)\.(.+)$`)

// ParseVersion parses `godot --version` output such as 4.7.2.stable.mono.official.abc123.
// The version is returned in release-tag form (4.7.2-stable, 4.8-beta1).
func ParseVersion(output string) (version string, dotnet bool, build string, err error) {
	m := versionRe.FindStringSubmatch(strings.TrimSpace(output))
	if m == nil {
		return "", false, "", fmt.Errorf("%w: %q", ErrUnknownVersionOutput, strings.TrimSpace(output))
	}
	build = "official"
	if strings.Contains(m[3], "custom_build") {
		build = "custom"
	}
	for _, p := range strings.Split(m[3], ".") {
		dotnet = dotnet || p == "mono"
	}
	return m[1] + "-" + m[2], dotnet, build, nil
}

// ProbeVersion runs `<path> --version` and parses the result.
func ProbeVersion(ctx context.Context, path string) (version string, dotnet bool, build string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return "", false, "", fmt.Errorf("run %s --version: %w", path, err)
	}
	return ParseVersion(string(out))
}
