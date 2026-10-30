# 0006. Git sources via archive URLs, https only

- Status: accepted
- Date: 2026-10-20
- Refs: [gd#13](https://github.com/godot-services/gd/issues/13)

## Context

Git assets could be fetched with `git clone` (any host, but requires the `git`
binary and a second install path) or as the archive zip the host provides.
The official asset library builds its download URLs the same way
(`src/Helpers/Utils.php`, `getComputedDownloadUrl`).

## Decision

`gd` turns `git` + `version` into an archive URL:
GitHub and Codeberg `<repo>/archive/<ref>.zip`, GitLab `<repo>/-/archive/<ref>.zip`.
Other hosts fail with "unsupported git host". The URL must use `https`, because
the first download of an asset has no hash to compare against. `version` is
required, since a default ref is not reproducible.

## Consequences

Library and Git assets share one install path (zip + SHA-256), no `git` binary
needed. Self-hosted Git servers are not supported. The ref is not URL-escaped yet
(deferred).
