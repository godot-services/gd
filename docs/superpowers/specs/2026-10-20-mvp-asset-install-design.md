# MVP v0.1.0: Asset Installation — Design

Status: draft for review · Date: 2026-10-20

## Goal

`gd` installs Godot assets into a project, reproducibly, from two sources:

- the official Godot Asset Library (by asset ID)
- a Git repository on GitHub, GitLab or Codeberg (by URL + ref)

Success: in a Godot project, `gd add <id>` installs an addon into `addons/`,
records it in `package.godot` and `package.lock`; on another machine
`gd install` reproduces exactly the same files or fails loudly; `gd remove`
undoes it.

## Context

- The official library (godotengine/godot-asset-library, `API.md`) knows only
  the **current** version of each asset. `GET /asset/{id}` returns one
  `download_url`, usually `<browse_url>/archive/<download_commit>.zip`.
- `download_commit` is often a branch (e.g. `master`), so the same URL can
  return different content over time. `download_hash` is always empty in the
  official library.
- Downloads are archives of the whole repository: one top-level folder,
  usually containing `addons/<name>/` plus demo files.

Consequence: reproducibility needs a lockfile that stores the SHA-256 of the
downloaded zip.

## Non-goals (MVP)

- custom registries (`registry=`), i.e. `gdar` — rejected with a clear error
- `gd search`, `update`, `outdated`
- separate handling of `dev_assets` (parsed, installed like `assets`)
- installing anything outside `addons/`
- Git hosts other than GitHub, GitLab, Codeberg; `git clone`
- editor version management

## Files

### `package.godot` (user-edited, INI)

```ini
[package]
name="My game"
version="1.0.0"

[assets]
dialogic={ library="1234" }
my_tool={ git="https://github.com/foo/tool", version="v1.2.0" }

[dev_assets]
gut={ library="1709" }
```

Per asset exactly one source key:

| key       | meaning                                   | `version`                       |
| --------- | ----------------------------------------- | ------------------------------- |
| `library` | official asset library asset ID           | informational, not used to resolve |
| `git`     | repository URL                            | ref (tag, branch, commit); required |
| `registry`| custom registry                           | rejected in MVP                 |

The nested section form (`[assets.my_tool]`) stays supported for reading.
`gd` writes the inline form.

Validation errors (unknown source, zero or several source keys, non-numeric
library ID) are returned as errors naming the asset.

### `package.lock` (written by `gd` only, INI)

```ini
[dialogic]
source="library:1234"
url="https://github.com/coppolaemilio/dialogic/archive/master.zip"
sha256="9f86d0…"
version_string="2.0"
paths="addons/dialogic"
```

- one section per asset name (names are unique across `assets` and `dev_assets`)
- `paths`: comma-separated top-level folders installed under `addons/`;
  the only folders `gd remove` ever deletes

## Commands

### `gd add <id|git-url> [--name <name>] [--version <ref>] [--dev]`

1. Use the current directory as project (`NewProject`; requires `project.godot`).
   A missing `package.godot` is created.
2. Resolve the source to a zip URL (see Sources).
3. Download, compute SHA-256, install (see Installation).
4. Write the asset to `package.godot` and the lock entry to `package.lock`.

Default name: library asset title slug (lowercase, `[a-z0-9_]`), or the
repository name for Git. Adding an existing name is an error.

### `gd install`

For every asset in `package.godot`:

- with lock entry: download `url`, compare SHA-256; mismatch → abort, no changes
- without lock entry: resolve like `add`, then write the lock entry

Lock entries without a matching asset are removed together with their `paths`.

### `gd remove <name>`

Delete the `paths` from the lock entry, then remove the asset from
`package.godot` and `package.lock`.

## Sources (`internal/source`)

- Library: `GET https://godotengine.org/asset-library/api/asset/{id}` →
  `download_url`, `title`, `version_string`. Non-200 or empty `download_url`
  → error.
- Git: compute the archive URL as the official library does
  (`src/Helpers/Utils.php`, `getComputedDownloadUrl`):
  - GitHub: `<repo>/archive/<ref>.zip`
  - GitLab: `<repo>/-/archive/<ref>.zip`
  - Codeberg: `<repo>/archive/<ref>.zip`
  - other hosts → error "unsupported git host"

A trailing `.git` and trailing `/` are stripped from the repository URL.

## Installation (`internal/install`)

1. Download into a temp dir under the OS temp dir; hash while writing.
2. If an expected hash is given and differs → error.
3. Open the zip. Strip the single top-level folder. Keep only entries under
   `addons/`. No `addons/` entries → error "asset contains no addons/ folder".
4. Reject every entry whose cleaned path is absolute or escapes the target
   (zip slip) → error, nothing installed.
5. Extract into a temp staging dir.
6. For each top-level folder `addons/<x>`: if it already exists in the project
   and is not in this asset's lock `paths` → error (never overwrite foreign
   files). Otherwise replace it with the staged folder.
7. Only after success the caller writes `package.godot` / `package.lock`.

## Code structure

| package           | responsibility                                  |
| ----------------- | ----------------------------------------------- |
| `internal/godot`  | `Package` (parse + write), `Lock` (parse + write), `Project` |
| `internal/source` | asset reference → zip URL + metadata            |
| `internal/install`| download + verify + extract `addons/**`         |
| `cmd`             | cobra commands `add`, `install`, `remove`       |

Library code returns errors and never prints or exits. Only `cmd` prints.

## Testing

- `internal/source`: `httptest` server faking `GET /asset/{id}`; table tests for
  Git URL → archive URL.
- `internal/install`: in-memory test zips: happy path, missing `addons/`, zip
  slip, hash mismatch, foreign folder conflict.
- `internal/godot`: round-trip test for writing `package.godot` (inline values
  and comments survive — known risk with `gopkg.in/ini.v1`), lock round-trip.
- E2E: `add` → `install` (fresh clone simulation) → `remove` in a temp project
  against `httptest`.
- CI: `go mod download`, `go vet ./...`, `go test ./... -cover`.

## Milestones

- **M0 Foundation** (gd): fix bugs from the analysis (no `os.Exit` in parser,
  `os.Stat` nil panics, `hasGodotProject`), rename `CODEOWNER` → `CODEOWNERS`,
  remove cobra boilerplate, CI cleanup, drop unused `test-coverage.yml`.
- **M1 MVP v0.1.0** (gd): this document.
- **M2 Comfort** (gd): `search`, `--no-dev`, `update`, `outdated`.
- **M3 Editor management** (gd): `gd editor list|install|uninstall|default`.
- **OpenAPI v1** (spec): OpenAPI description of the library API.
- **Read-only registry** (gdar): fix `/api/api/asset` routing, implement
  `configure`, `asset` list/get; then `registry=` support in gd.
