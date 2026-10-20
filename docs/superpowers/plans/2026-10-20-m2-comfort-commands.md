# M2 Comfort Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `gd search`, `gd install --no-dev`, `gd outdated` and `gd update`.

**Architecture:** `internal/godot` reads the Godot version from `project.godot`, `internal/source` gains a list/search call, `cmd` gets three new commands and a flag. `install` and `update` share one helper that installs an asset and saves its lock entry.

**Tech Stack:** Go (version from `go.mod`), cobra, `gopkg.in/ini.v1`, stdlib (`text/tabwriter`, `net/http/httptest`, `regexp`).

**Spec:** `docs/superpowers/specs/2026-10-20-m2-comfort-commands-design.md`

## Global Constraints

- No new dependencies. Library code never prints or exits.
- Search request: `GET {base}/asset?type=addon&filter=<text>&godot_version=<v>&max_results=<n>`; `--max` default 20, allowed 1–500.
- Godot version: `--godot` flag, else first quoted `^\d+\.\d+(\.\d+)?$` entry of the `config/features=PackedStringArray(...)` line in `project.godot`.
- `outdated` exits 1 (returns an error) when any asset is outdated; it never writes files.
- `update` never changes `package.godot`.
- Tests: stdlib `testing`, same package, `cmd` tests not parallel and use the existing helpers (`newTestProject`, `newFakeLibrary`, `run`, `exists`, `isolateCache`).
- Commit messages end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` and reference the issue (`Refs #N` / `Closes #N`).

## Review Focus

1. `config/features` whose first entry is not a version (`PackedStringArray("Forward Plus", "4.3")`) → `4.3`. Test: Task 1 `TestGodotVersion` case "version not first".
2. `gd search --godot 4.3` outside any Godot project → works. Test: Task 3 `TestSearchOutsideProjectWithGodotFlag`.
3. An asset was deleted from the library (404) → `gd outdated` fails with that error instead of printing wrong rows. Test: Task 5 `TestOutdatedLibraryError`.
4. `gd install --no-dev` with a dev asset that has no lock entry → no request for it at all. Test: Task 4 `TestInstallNoDevSkipsDevAssets` (request counter).
5. `gd update` of an asset whose new version ships a different addon folder → old folder removed, new one installed. Test: Task 6 `TestUpdateReplacesChangedFolders`.

---

### Task 1: Godot version from project.godot (#35)

**Files:**
- Modify: `internal/godot/project.go`, `internal/godot/project_test.go`

**Interfaces:**
- Produces: `func (p *Project) GodotVersion() (string, error)`; `var ErrNoGodotVersion = errors.New("no Godot version in project.godot, pass --godot")`.

- [ ] **Step 1: Write failing table test `TestGodotVersion`** (writes `project.godot` into a temp dir, `NewProject`, `GodotVersion`)

```go
cases := []struct{ name, src, want string; err error }{
	{"godot 4", "[application]\nconfig/name=\"G\"\nconfig/features=PackedStringArray(\"4.3\", \"Forward Plus\")\n", "4.3", nil},
	{"patch", "[application]\nconfig/features=PackedStringArray(\"4.2.1\")\n", "4.2.1", nil},
	{"version not first", "[application]\nconfig/features=PackedStringArray(\"Forward Plus\", \"4.3\")\n", "4.3", nil},
	{"godot 3", "[application]\nconfig/name=\"G\"\n", "", ErrNoGodotVersion},
	{"no version entry", "[application]\nconfig/features=PackedStringArray(\"Mobile\")\n", "", ErrNoGodotVersion},
	{"multi-line values around", "[input]\nui_x={\n\"deadzone\": 0.5,\n\"events\": []\n}\n[application]\nconfig/features=PackedStringArray(\"4.4\")\n", "4.4", nil},
}
```

- [ ] **Step 2:** `go test ./internal/godot -run TestGodotVersion -v` → FAIL (undefined).
- [ ] **Step 3:** Implement with two package-level regexps: line `(?m)^config/features=PackedStringArray\(([^)]*)\)` and entries `"([^"]*)"`; return the first entry matching `^\d+\.\d+(\.\d+)?$`.
- [ ] **Step 4:** `go test ./internal/godot -v` → PASS.
- [ ] **Step 5:** Commit `Read Godot version from project.godot` + `Refs #35`.

### Task 2: Library search (#35)

**Files:**
- Modify: `internal/source/source.go`, `internal/source/source_test.go`

**Interfaces:**
- Produces:

```go
type SearchResult struct {
	AssetID, Title, VersionString, GodotVersion, Cost, SupportLevel string
}
func (l Library) Search(ctx context.Context, query, godotVersion string, max int) ([]SearchResult, int, error)
```

  JSON keys: `asset_id`, `title`, `version_string`, `godot_version`, `cost`, `support_level`; the `int` is `total_items`.

- [ ] **Step 1: Write failing tests**
  - `TestLibrarySearch`: httptest handler for `/asset` asserts `type=addon`, `filter=gut`, `godot_version=4.3`, `max_results=5`, returns two results with `total_items: 7`; want both mapped results and total 7.
  - `TestLibrarySearchError`: handler returns 500 → error.
- [ ] **Step 2:** `go test ./internal/source -run TestLibrarySearch -v` → FAIL.
- [ ] **Step 3:** Implement; build the query with `url.Values`.
- [ ] **Step 4:** `go test ./internal/source -v` → PASS.
- [ ] **Step 5:** Commit `Add asset library search` + `Refs #35`.

### Task 3: `gd search` (#35)

**Files:**
- Create: `cmd/search.go`, `cmd/search_test.go`
- Modify: `cmd/root.go`, `cmd/helpers_test.go`

**Interfaces:**
- Consumes: `godot.NewProject`, `Project.GodotVersion`, `source.Library.Search`.
- Produces: `func newSearchCmd(opts *options) *cobra.Command`; test helper extension: `fakeLibrary` serves `GET /asset` from a field `results []map[string]string` and records the last query in `lastQuery url.Values`.
- Output: `text/tabwriter` with header `ID\tTITLE\tVERSION\tGODOT\tLICENSE\tSUPPORT`, then `%d of %d results. Install with: gd add <ID>`; empty → `no assets found for "<text>" (Godot <v>)`.
- Helper `runOut(t, lib, args...) (string, error)` in `helpers_test.go` that captures `cmd.SetOut` output (keep `run` as is).

- [ ] **Step 1: Write failing tests**
  - `TestSearchUsesProjectGodotVersion`: project with `config/features=PackedStringArray("4.3")`, `search gut` → `lastQuery.Get("godot_version") == "4.3"`, output contains the result title and `1 of 1 results`.
  - `TestSearchGodotFlagOverrides`: same project, `--godot 4.2` → `4.2`.
  - `TestSearchOutsideProjectWithGodotFlag`: temp dir without `project.godot`, `--godot 4.3` → no error.
  - `TestSearchOutsideProjectWithoutGodotFlag`: → `errors.Is(err, godot.ErrNoGodotProject)`.
  - `TestSearchInvalidMax`: `--max 0` and `--max 501` → error, no request (`lastQuery == nil`).
  - `TestSearchNoResults`: empty results → output contains `no assets found`, err nil.
- [ ] **Step 2:** `go test ./cmd -run TestSearch -v` → FAIL.
- [ ] **Step 3:** Implement; register in `root.go`.
- [ ] **Step 4:** `go test ./... ` → PASS.
- [ ] **Step 5:** Commit `Add gd search` + `Closes #35`.

### Task 4: shared install helper and `--no-dev` (#36)

**Files:**
- Modify: `cmd/install.go`, `cmd/assets.go`, `cmd/install_test.go`, `cmd/helpers_test.go`

**Interfaces:**
- Produces in `cmd/assets.go`:

```go
// installAndLock installs one asset and saves its lock entry right away.
func installAndLock(ctx context.Context, lib source.Library, dir string, lock godot.Lock, name string, a godot.PackageAssetConfiguration, entry *godot.LockEntry) (godot.LockEntry, error)
```

- `fakeLibrary` gets `assetHits int` (requests to `/asset/1234`).
- `gd install --no-dev`: loop over `pkg.Assets` only (sorted); stale cleanup skips names in `pkg.DevAssets`. Without the flag: unchanged (`AllAssets`).

- [ ] **Step 1: Write failing test `TestInstallNoDevSkipsDevAssets`**: `package.godot` with `[assets] run={ library="1234" }` and `[dev_assets] tool={ library="1234" }`; lock entry for `tool` with `Paths: ["addons/devtool"]` and folder `addons/devtool` on disk. `install --no-dev` → `addons/x` installed, `addons/devtool` still exists, lock still has `tool` unchanged, `assetHits == 1` (only `run` resolved).
- [ ] **Step 2:** `go test ./cmd -run TestInstallNoDev -v` → FAIL (unknown flag).
- [ ] **Step 3:** Extract `installAndLock` from the install loop (behaviour unchanged), then add the flag.
- [ ] **Step 4:** `go test ./...` → PASS (existing install tests prove the refactor).
- [ ] **Step 5:** Commit `Add gd install --no-dev` + `Closes #36`.

### Task 5: `gd outdated` (#37)

**Files:**
- Create: `cmd/outdated.go`, `cmd/outdated_test.go`
- Modify: `cmd/root.go`, `cmd/helpers_test.go`

**Interfaces:**
- Consumes: `source.Resolve`, `godot.LoadLock` via `loadProject`.
- Produces: `var errOutdated = errors.New("assets are outdated")`, returned as `fmt.Errorf("%d %w", n, errOutdated)`; `func newOutdatedCmd(opts *options) *cobra.Command`.
- `fakeLibrary` fields `version string` (default `"2.0"`) used in `/asset/1234` JSON as `version_string` and in `download_url` as `%s/zip?v=<version>`, so a version change also changes the URL.
- Rows (tabwriter, header `NAME\tCURRENT\tLATEST\tSOURCE`): outdated library asset `name  2.0  2.1  library:1234`; git asset `name  <ref>  pinned  git:<repo>@<ref>`; no lock entry `name  -  not installed  <source id>`. Nothing to show → `all assets are up to date`.

- [ ] **Step 1: Write failing tests**
  - `TestOutdatedReportsNewVersion`: `add 1234`, set `lib.version = "2.1"` → `errors.Is(err, errOutdated)`, output has `2.0` and `2.1`.
  - `TestOutdatedUpToDate`: after `add` → err nil, output `all assets are up to date`.
  - `TestOutdatedShowsPinnedAndNotInstalled`: `package.godot` with a git asset and a library asset without lock → err nil, output contains `pinned` and `not installed`.
  - `TestOutdatedLibraryError`: `add`, then make `/asset/1234` return 404 (field `assetStatus`) → error, not `errOutdated`.
  - `TestOutdatedWritesNothing`: modification times / contents of `package.godot` and `package-lock.godot` unchanged after an outdated run.
- [ ] **Step 2:** `go test ./cmd -run TestOutdated -v` → FAIL.
- [ ] **Step 3:** Implement; register in `root.go`.
- [ ] **Step 4:** `go test ./...` → PASS.
- [ ] **Step 5:** Commit `Add gd outdated` + `Closes #37`.

### Task 6: `gd update` (#38)

**Files:**
- Create: `cmd/update.go`, `cmd/update_test.go`
- Modify: `cmd/root.go`

**Interfaces:**
- Consumes: `installAndLock` (Task 4), `loadProject`.
- Produces: `func newUpdateCmd(opts *options) *cobra.Command`. Per asset: `installAndLock(..., &godot.LockEntry{Paths: old.Paths})`; print `updated <name> <old.VersionString> -> <new.VersionString>` when URL or SHA256 changed, else `<name> is up to date`.

- [ ] **Step 1: Write failing tests**
  - `TestUpdateAll`: `add 1234`; `lib.version = "2.1"`, `lib.zip` with new content → `update` → lock `VersionString == "2.1"`, new SHA, new file content on disk, `package.godot` bytes unchanged.
  - `TestUpdateNamedOnly`: `add 1234 --name a`; then append a git asset `b` to `package.godot` with a hand-written lock entry (the same library zip cannot be added twice, both would own `addons/x`) → `update a` → `assetHits` increased by one, `b` lock entry unchanged (no request to GitHub).
  - `TestUpdateUnknownName`: `update nope` → `errors.Is(err, godot.ErrAssetNotFound)`, `assetHits == 0`.
  - `TestUpdateUpToDate`: `add`, `update` → output `is up to date`.
  - `TestUpdateReplacesChangedFolders`: `add`; new zip ships `addons/y` instead of `addons/x` → `update` → `addons/x` gone, `addons/y` present, lock paths `[addons/y]`.
- [ ] **Step 2:** `go test ./cmd -run TestUpdate -v` → FAIL.
- [ ] **Step 3:** Implement; register in `root.go`.
- [ ] **Step 4:** `go test ./...` → PASS.
- [ ] **Step 5:** Commit `Add gd update` + `Refs #38`.

### Task 7: ADR 0015 and README (#38)

**Files:**
- Create: `docs/adr/0015-comfort-commands.md`
- Modify: `docs/adr/README.md`, `README.md`, spec status line

- [ ] **Step 1:** ADR 0015 (template from `docs/adr/template.md`): Godot version source, `--no-dev` semantics and cleanup exception, meaning of outdated, exit code 1, git refs stay pinned, `update` does not touch `package.godot`. Add to index.
- [ ] **Step 2:** README usage block: `gd search`, `gd outdated`, `gd update`, `gd install --no-dev`; roadmap rows for the four commands → `done`.
- [ ] **Step 3:** `go vet ./... && go test -race ./...` → PASS.
- [ ] **Step 4:** Commit `Document comfort commands (ADR 0015)` + `Closes #38`.
