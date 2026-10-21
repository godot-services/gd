# M4 Release Builds — Design

Status: accepted · Date: 2026-10-21 · Refs: gd#41 · Builds on: [ADRs](../../adr/README.md)

## Goal

Every GitHub release of `gd` ships prebuilt binaries, so users do not need a Go
toolchain:

- linux, macOS and windows, each on amd64 and arm64;
- one archive per target plus `checksums.txt` (SHA-256);
- built automatically when a `v*` tag is pushed;
- `gd --version` prints the release version.

The package managers of gd#42 consume these archives later.

## Context

- Today users install with `go install github.com/godot-services/gd@vX`; that
  keeps working.
- Releases v0.1.0–v0.3.0 exist with hand-written notes, created together with
  their tag (`gh release create vX --notes ...`). The pipeline must not replace
  those notes.
- `go.mod` says `go 1.23.2`; CI installs that version. Go only stamps the VCS
  tag into `debug.ReadBuildInfo` for `go build` from 1.24 on, so a release build
  cannot rely on BuildInfo alone. `go install module@vX` does fill
  `Main.Version`.
- `gd` has OS-specific code (editor paths, detached start), so cross-compiling
  every target is worth checking before a tag, not only at release time.

## Decisions

| Question | Decision | ADR |
| --- | --- | --- |
| Tooling | GoReleaser v2, run by `goreleaser/goreleaser-action` | 0020 |
| Release flow | `release.mode: keep-existing`: assets are added to an existing release; a bare tag gets a release with a generated changelog | 0020 |
| Signing | GitHub artifact attestation (Sigstore, keyless) of the archives; no macOS notarization, no Windows code signing | 0021 |
| Version | `-ldflags -X` in release builds, `debug.ReadBuildInfo` fallback, else `dev` | 0022 |

Rejected: a hand-written `go build` matrix (more YAML and shell, and gd#42 would
have to generate formulas and manifests itself); notarization and code signing
(needs a paid Apple Developer ID and a Windows certificate, plus secrets; the
package managers of gd#42 avoid most Gatekeeper/SmartScreen prompts anyway).

## Design

### `.goreleaser.yaml`

- `version: 2`.
- One build: `main` package at the repository root, binary `gd`,
  `CGO_ENABLED=0`, `-trimpath`, `ldflags: -s -w -X
  github.com/godot-services/gd/cmd.buildVersion=v{{.Version}}`; `goos` linux, darwin,
  windows; `goarch` amd64, arm64 (6 targets).
- Archives named `gd_{{.Version}}_{{.Os}}_{{.Arch}}`, `tar.gz`, `zip` for
  windows, containing the binary, `README.md` and `LICENSE`.
- `checksum.name_template: checksums.txt`, SHA-256.
- `release.mode: keep-existing`.
- `changelog.use: github` (only shown when GoReleaser creates the release).

`{{.Version}}` is the tag without the leading `v` (`0.4.0`); the ldflag gets
`v{{.Version}}` so `gd --version` matches the tag and `go install` output.

### `.github/workflows/release.yml`

- Trigger: `push` of tags `v*`.
- Permissions: `contents: write` (release upload), `id-token: write` and
  `attestations: write` (attestation).
- Steps: checkout with `fetch-depth: 0` (changelog), `setup-go` from `go.mod`,
  `goreleaser/goreleaser-action` with a pinned GoReleaser version and
  `release --clean`, then `actions/attest-build-provenance` with
  `subject-path: dist/*.tar.gz, dist/*.zip`.
- Only `GITHUB_TOKEN`; no new secrets.

### PR check in `.github/workflows/test.yml`

A second job runs `goreleaser release --snapshot --clean` with the same pinned
version. It builds all 6 targets and the archives without publishing, so a
broken config or a target that does not compile fails the PR.

### Version in the binary: `cmd/version.go`

```go
var buildVersion string // set by -ldflags -X in release builds

func resolveVersion(ldflags string, info *debug.BuildInfo, ok bool) string
```

1. `ldflags` non-empty → use it.
2. Else BuildInfo available and `Main.Version` neither empty nor `(devel)` →
   use it (`go install …@v0.4.0` gives `v0.4.0`).
3. Else `dev`.

`newRootCmd` sets `rootCmd.Version = resolveVersion(buildVersion, debug.ReadBuildInfo())`;
cobra adds `--version` / `-v` to the root command only, so `gd add --version`
is unaffected, and prints `gd version v0.4.0`. The variable is named
`buildVersion` because `cmd/add.go` already uses a local `version`. No
`version` subcommand (YAGNI).

### Documentation

- ADR 0020, 0021 and 0022 (next free numbers when committing; renumbered if the
  parallel M7 or M5 work took them), index in `docs/adr/README.md`.
- README "Installation": download from the releases page, verify with
  `sha256sum -c checksums.txt --ignore-missing` and
  `gh attestation verify <archive> --repo godot-services/gd`, macOS hint
  `xattr -d com.apple.quarantine gd` for the unsigned binary; `go install`
  stays as the second option.

## Error handling

- A failed build or upload fails the release job; the tag stays, the user
  re-runs the job (`--clean` and `keep-existing` make re-runs safe).
- An existing release with assets of the same name: GoReleaser replaces them
  (`replace_existing_artifacts: true`) so a re-run after a partial upload
  works.

## Testing

- `cmd/version_test.go` (TDD, each test seen failing first): table test of
  `resolveVersion` for ldflags set, BuildInfo with a version, BuildInfo
  `(devel)`, no BuildInfo; and a test that `gd --version` prints
  `gd version <version>`.
- Local snapshot: `goreleaser release --snapshot --clean` (run via `go run`, no
  install) produces 6 archives and `checksums.txt`; `sha256sum -c` passes; the
  darwin/arm64 binary prints the snapshot version.
- The tag-triggered upload and attestation can only run on a real tag; they are
  first exercised by the next release (v0.4.0 or later).

## Out of scope

Package managers (gd#42), `gd self-update`, SBOMs, Docker images.
