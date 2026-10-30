# 0023. Package managers: Homebrew tap and Scoop bucket

- Status: accepted
- Date: 2026-10-21
- Refs: [gd#42](https://github.com/godot-services/gd/issues/42), [gd#74](https://github.com/godot-services/gd/issues/74), builds on [0020](0020-release-builds-with-goreleaser.md)

## Context

Release archives exist since [0020](0020-release-builds-with-goreleaser.md).
Users expect `brew install` on macOS and Linux and a Windows package manager.
GoReleaser can write a Homebrew cask (`homebrew_casks`; formulas are deprecated
in v2) and a Scoop manifest from the same config. Writing to other
repositories needs a token; `GITHUB_TOKEN` only covers `gd`. The binaries are
not notarized ([0021](0021-build-attestation-instead-of-signing.md)), and
Homebrew quarantines cask downloads on macOS.

## Decision

- First: a Homebrew cask in `godot-services/homebrew-tap` (`Casks/gd.rb`,
  macOS and Linux) and a Scoop manifest in `godot-services/scoop-bucket`
  (`bucket/gd.json`).
- GoReleaser writes both on every release; nobody edits them by hand. Tags
  with a prerelease part skip both (`skip_upload: auto`) and are marked as
  prerelease on GitHub (`release.prerelease: auto`).
- Credentials: a fine-grained PAT scoped to the two repositories with
  "Contents: read and write", stored as the `gd` secret `TAP_GITHUB_TOKEN`.
- The cask removes the quarantine attribute from the binary on macOS with a
  `postflight_steps` `run` step, emitted through GoReleaser's `custom_block`.
  GoReleaser's own `hooks.post.install` renders a `postflight` block, which
  Homebrew 7 deprecates at runtime (a warning for every user, an error with
  `HOMEBREW_DEVELOPER=1`, later an error for everyone).
- Archives are reproducible (`mod_timestamp`, archive `mtime` from the commit),
  so a re-run uploads the same SHA-256 the tap and bucket already contain.
- Only the latest stable tag is ever re-run: a re-run of an older tag would
  rewrite the cask and manifest to that version and downgrade users.

Rejected for now: winget (PRs to `microsoft/winget-pkgs` need a fork and a
manual first review), deb/rpm, AUR and Nix (all in [gd#74](https://github.com/godot-services/gd/issues/74)); a GitHub App instead
of the PAT (more setup for two repositories, can replace the PAT later without
changing the config shape).

## Consequences

A stable tag updates Homebrew and Scoop with no manual step. The PAT expires
and belongs to a person: when it lapses, the release and its attestation still
publish but the tap/bucket push fails until the secret is renewed and the job
re-run. `postflight_steps` needs Homebrew 7; older Homebrew cannot load the
cask (`brew update` fixes it). `brew style` reports stanza-order offenses in the
generated cask; they only matter for homebrew-core. The cask shares its name
with homebrew-core's `gd` formula (libgd), so `brew install` needs the full
name `godot-services/tap/gd`.
