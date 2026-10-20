# 0008. gd only touches folders it installed

- Status: accepted, amended by [0013](0013-hardening-after-mvp-audit.md)
- Date: 2026-10-20
- Refs: gd#14, gd#15

## Context

Archives come from third parties and the user's project contains their own work.
Data loss or writes outside the project must be impossible.

## Decision

- The lockfile records the installed top-level folders per asset (`paths`).
  `gd` only replaces or deletes folders listed there.
- An existing `addons/<name>` not owned by the asset aborts the install
  ("folder exists and is not managed by this asset").
- If any archive entry escapes the target (absolute path, `..`), the whole
  archive is rejected before anything is written (zip slip).
- Extraction goes to a staging folder inside the project first; folders are
  swapped in only after every check passed.
- `gd remove` and stale-entry cleanup accept only clean `addons/<name>` paths.

## Consequences

Safe by default; a manually copied addon blocks installing the same addon until
it is deleted. No size limit for archive entries yet (`shortcut:` comment in
`internal/install`). Dangling symlinks in `addons/` are not detected yet (deferred).
