# 0015. Comfort commands: search, --no-dev, outdated, update

- Status: accepted
- Date: 2026-10-20
- Refs: gd#17, gd#35, gd#36, gd#37, gd#38, spec `docs/superpowers/specs/2026-10-20-m2-comfort-commands-design.md`

## Context

After v0.1.0 users had to find asset IDs on the website, could not skip dev
tools in CI, and had no way to see or apply new versions. The asset library
only serves the latest version of an asset (ADR 0004), and git assets are pinned
to a ref (ADR 0006).

## Decision

- **Godot version for search:** `--godot`, else the first `X.Y[.Z]` entry of
  `config/features=PackedStringArray(...)` in `project.godot`. Only that line is
  read, because `project.godot` can hold multi-line values `ini.v1` may not
  parse. Godot 3 projects have no such entry and need `--godot`. A version is
  always sent, because without it the library lists only Godot 2.1 assets.
- **`gd search`** lists `type=addon` only, since `gd` installs only addons;
  `--max` is 1–500 (library limit), default 20.
- **`gd install --no-dev`** skips dev assets entirely: no resolve, download,
  install or removal. Lock entries of dev assets are never treated as stale.
- **Outdated** means: the library now returns another `download_url` or
  `version_string` than the lock entry. Git assets are shown as `pinned`; `gd`
  does not query git hosts for newer tags. When the source in `package.godot`
  differs from the lock entry, the row says `run gd install` instead of
  comparing unrelated sources.
- **`gd outdated` exits with 1** when any asset is outdated (like `npm
  outdated`), so CI can fail on it. It never writes files.
- **`gd update [name...]`** re-resolves assets (library: latest; git: same ref,
  so only branch refs get new content), keeps ownership of the installed
  folders and saves the lock after each asset. It never changes `package.godot`.
  It reports `updated` when URL, version string or hash changed, so it agrees
  with `gd outdated`.

## Consequences

Day-to-day work without the website. Raising a git ref stays a manual edit of
`version` in `package.godot` followed by `gd install`. `gd outdated` needs one
library request per library asset.
