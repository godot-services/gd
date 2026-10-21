package cmd

import "runtime/debug"

// buildVersion is set by -ldflags -X in release builds (.goreleaser.yaml).
var buildVersion string

// resolveVersion prefers the release build's version, then the module version
// go install records, and falls back to "dev" for local builds.
func resolveVersion(ldflags string, info *debug.BuildInfo, ok bool) string {
	if ldflags != "" {
		return ldflags
	}
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
