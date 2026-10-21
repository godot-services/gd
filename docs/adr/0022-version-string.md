# 0022. Version string in the binary

- Status: accepted
- Date: 2026-10-21
- Refs: gd#41

## Context

`gd --version` must print the release version. Release archives are built with
`go build` from a tagged checkout; `go.mod` says Go 1.23, and CI builds with
that version, which does not stamp the VCS tag into `debug.ReadBuildInfo` for
`go build` (that came in Go 1.24).
`go install github.com/godot-services/gd@vX` does record `vX` as the main
module version.

## Decision

- Release builds set `cmd.buildVersion` with `-ldflags -X` to `v<version>`
  (see [0020](0020-release-builds-with-goreleaser.md)).
- Without it, `gd` uses `debug.ReadBuildInfo().Main.Version`, unless it is empty
  or `(devel)`.
- Otherwise the version is `dev`.
- cobra's built-in flag prints it: `gd --version` / `gd -v` →
  `gd version v0.4.0`. No `gd version` subcommand.

## Consequences

Release archives and `go install` print the same tag. `go run` prints `dev`;
a local `go build` with Go 1.24 or later prints the pseudo-version Go stamps
(e.g. `v0.3.1-0.20261021112817-8eaf21562ced`, `+dirty` for a dirty tree), which
names the commit and is kept on purpose.
The flag lives on the root command only, so `gd add --version <ref>` keeps its
meaning. Once `go.mod` moves to Go 1.24 or later, the ldflag could be dropped.
