# M2 Comfort Commands — Design

Status: draft for review · Date: 2026-10-20 · Refs: gd#17 · Builds on: [MVP spec](2026-10-20-mvp-asset-install-design.md), [ADRs](../../adr/README.md)

## Goal

Four additions that make `gd` usable day to day:

- `gd search` finds asset IDs without opening the website.
- `gd install --no-dev` installs only runtime assets (CI, exports).
- `gd outdated` shows which library assets have a newer version, and fails CI when any do.
- `gd update` moves assets to their latest version and re-locks them.

## Context

- `GET /asset?…` lists assets. Without `godot_version` the official library
  returns **only Godot 2.1 assets** (legacy behaviour), so `gd` must always send one.
- Godot 4 writes the engine version into `project.godot`:
  `config/features=PackedStringArray("4.3", "Forward Plus")` in `[application]`.
  Godot 3 projects have no such entry. `project.godot` can contain multi-line
  values that `gopkg.in/ini.v1` may not parse, so only this one line is read.
- The library serves only the latest version of an asset (ADR 0004). A library
  asset is outdated when the library now returns a different `download_url` or
  `version_string` than the lock entry.
- Git assets are pinned to their ref (ADR 0006). Only a branch ref can get new
  content.
- `gd install` removes lock entries without asset (ADR 0011).

## Non-goals

- Searching or updating git assets via host APIs (newer tags, semver).
- Removing already installed dev assets with `--no-dev` (like `npm prune`).
- Paging through search results beyond `--max`.
- Searching custom registries.

## Godot version (`internal/godot`)

`func (p *Project) GodotVersion() (string, error)`

- Reads `project.godot` from `p.Location` and finds the line
  `config/features=PackedStringArray(...)`.
- Returns the first quoted entry matching `^\d+\.\d+(\.\d+)?$`, e.g. `4.3`.
- No such line or entry → `ErrNoGodotVersion`
  ("no Godot version in project.godot, pass --godot").

## `gd search <text> [--godot <version>] [--max <n>]`

- Godot version: `--godot` if set; else `NewProject(".")` + `GodotVersion()`.
  Outside a project and without `--godot` → the error from `NewProject` /
  `GodotVersion`, wrapped with "pass --godot".
- `--max` default 20, allowed 1–500 (library limit); other values → error.
- Request: `GET {base}/asset?type=addon&filter=<text>&godot_version=<v>&max_results=<n>`.
- `internal/source`:

  ```go
  type SearchResult struct {
      AssetID, Title, VersionString, GodotVersion, Cost, SupportLevel string
  }
  func (l Library) Search(ctx context.Context, query, godotVersion string, max int) ([]SearchResult, int, error)
  ```

  The `int` is `total_items`. Non-200 → error with status.
- Output (`text/tabwriter`):

  ```
  ID    TITLE                VERSION  GODOT  LICENSE  SUPPORT
  1709  GUT - Godot Unit…    9.6.1    4.3    MIT      community
  3 of 3 results. Install with: gd add <ID>
  ```

  No results → `no assets found for "<text>" (Godot <v>)`, exit 0.

## `gd install --no-dev`

- Assets in `dev_assets` are skipped: not resolved, downloaded, installed or removed.
- Stale-entry cleanup (ADR 0011) skips lock entries whose name is a dev asset,
  so their entries and folders stay.
- Without the flag nothing changes.

## `gd outdated`

For every asset in `package.godot`, sorted by name:

| case | CURRENT | LATEST | status |
| --- | --- | --- | --- |
| library asset, lock entry, library returns other `download_url` or `version_string` | lock `version_string` | library `version_string` | outdated |
| library asset, lock entry, same values | lock `version_string` | same | up to date (not printed) |
| git asset | ref | `pinned` | not printed as outdated |
| asset without lock entry | `-` | `not installed` | not outdated |

- Output (`text/tabwriter`): `NAME  CURRENT  LATEST  SOURCE`, one row per asset
  that is outdated, pinned or not installed. All up to date → `all assets are up to date`.
- Any outdated asset → command returns `errOutdated` ("N assets are outdated"),
  so `gd` exits with 1 (like `npm outdated`). Otherwise exit 0.
- Library errors (network, 404) abort with that error.
- Only reads; never writes files.

## `gd update [name...]`

- Without names: all assets in `package.godot`. With names: only these; any
  unknown name → `godot.ErrAssetNotFound` before any network call.
- Per asset (sorted by name): resolve again (library: latest; git: same ref),
  download, extract with the asset's locked `paths` as owned, remove folders the
  asset no longer ships, save the lock after each asset. This is the existing
  `installAsset` with a lock entry that keeps only `Paths`.
- Output per asset: `updated <name> <old> -> <new>` when URL or hash changed,
  else `<name> is up to date`.
- `package.godot` is not changed.

## Code structure

| file | change |
| --- | --- |
| `internal/godot/project.go` | `GodotVersion`, `ErrNoGodotVersion` |
| `internal/source/source.go` | `SearchResult`, `Library.Search` |
| `cmd/search.go` | new |
| `cmd/outdated.go` | new, `errOutdated` |
| `cmd/update.go` | new |
| `cmd/install.go` | `--no-dev`; per-asset install + lock save + print shared with `update` |
| `cmd/root.go` | register commands |

## Testing

Against the existing `httptest` fake library, extended with:

- `GET /asset` list handler that asserts `type=addon`, `filter`, `godot_version`
  and `max_results`;
- mutable title, version string and zip, to simulate a new upstream version.

Tests:

- `GodotVersion`: Godot 4 file (`"4.3"`), patch version (`"4.2.1"`), no features
  line (Godot 3), features without version, surrounding multi-line values.
- `Search`: query parameters, result mapping, non-200.
- `gd search`: version from `project.godot`, `--godot` overrides, outside project
  without `--godot` fails, invalid `--max`, no results message.
- `gd install --no-dev`: dev asset not downloaded, existing dev folder and lock
  entry untouched; runtime asset installed.
- `gd outdated`: exit error when library changed, nil when not, git asset shown
  as pinned, missing lock entry shown as not installed, no file written.
- `gd update`: all assets, named asset only, unknown name fails without request,
  new hash in lock and new files on disk, up-to-date message.

## Docs

- ADR 0015 "Comfort commands": Godot version source, `--no-dev` semantics and
  cleanup exception, meaning of outdated, exit code, git refs stay pinned.
- README usage and roadmap table; `docs/configuration-files.md` unchanged.

## GitHub

Split epic gd#17 into four issues in milestone M2 (search, install --no-dev,
outdated, update), each referencing the epic. Close the epic when all are done.
