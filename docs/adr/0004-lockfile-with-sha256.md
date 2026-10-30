# 0004. Lockfile with SHA-256 per asset

- Status: accepted, amended by [0013](0013-hardening-after-mvp-audit.md)
- Date: 2026-10-20
- Refs: [gd#12](https://github.com/godot-services/gd/issues/12)

## Context

The official asset library knows only the current version of an asset.
`download_commit` is often a branch such as `master`, so the same download URL
returns different content over time, and `download_hash` is always empty.
`version` in `package.godot` can therefore not be resolved against the library.

## Decision

`gd` writes a lockfile with the resolved download URL, the SHA-256 of the
downloaded zip, the version string and the installed folders per asset.
`gd install` downloads exactly that URL and aborts on a hash mismatch.
For library assets `version` in `package.godot` is informational only.

## Consequences

Reproducible installs, and a changed upstream archive is detected instead of
silently installed. When upstream moves a branch, `gd install` fails until the
asset is added again. A lock entry with an empty hash is currently installed
unverified (deferred).
