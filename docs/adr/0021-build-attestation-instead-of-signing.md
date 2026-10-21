# 0021. Build attestation instead of code signing

- Status: accepted
- Date: 2026-10-21
- Refs: gd#41

## Context

Unsigned binaries trigger Gatekeeper on macOS (quarantined downloads) and
SmartScreen on Windows. Notarization needs a paid Apple Developer ID, a Windows
code-signing certificate costs money too, and both need secrets in CI. Users
still want to check that an archive was built from this repository.

## Decision

- No macOS notarization and no Windows code signing.
- The release workflow creates a GitHub artifact attestation (Sigstore,
  keyless, via the workflow's OIDC token) for every archive with
  `actions/attest-build-provenance`; users check it with
  `gh attestation verify <archive> --repo godot-services/gd`.
- `checksums.txt` (SHA-256) covers integrity without `gh`.
- The README tells macOS users to drop the quarantine flag
  (`xattr -d com.apple.quarantine gd`).

## Consequences

No accounts, costs or secrets. Users downloading by hand see the OS warnings;
the package managers of gd#42 avoid most of them. If signing becomes necessary,
a new ADR adds it on top of the attestation.
