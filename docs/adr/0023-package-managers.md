# 0023. Package managers: Homebrew tap and Scoop bucket

- Status: accepted
- Date: 2026-10-21
- Refs: gd#42, gd#74, builds on [0020](0020-release-builds-with-goreleaser.md)

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
- The cask's post-install hook removes the quarantine attribute from the
  binary on macOS.

Rejected for now: winget (PRs to `microsoft/winget-pkgs` need a fork and a
manual first review), deb/rpm, AUR and Nix (all in gd#74); a GitHub App instead
of the PAT (more setup for two repositories, can replace the PAT later without
changing the config shape).

## Consequences

A stable tag updates Homebrew and Scoop with no manual step. The PAT expires
and belongs to a person: when it lapses, the release itself still publishes
but the tap/bucket push fails until the secret is renewed and the job re-run.
The generated cask uses `postflight`, which Homebrew's own style check
(`brew style`) flags for homebrew-core; third-party taps install it fine.
