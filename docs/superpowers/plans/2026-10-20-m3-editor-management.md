# M3 Editor Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `gd` installs and registers Godot 4 editors and export templates, resolves the project's editor from `package.godot`, and opens/runs the project with it.

**Architecture:** `internal/godot` gets the `[editor]` spec, `package.local.godot` and the main scene; new `internal/godotbuilds` knows the release layout of `godotengine/godot-builds`; new `internal/editor` owns data dirs, the registry, resolution, editor/template installation and starting processes. `cmd` adds `editor`, `open`, `start`, `godot` and `templates`.

**Tech Stack:** Go (version from `go.mod`), cobra, `gopkg.in/ini.v1`, stdlib (`archive/zip`, `crypto/sha512`, `os/exec`, `net/http/httptest`, `syscall` via build tags).

**Spec:** `docs/superpowers/specs/2026-10-20-m3-editor-management-design.md`

## Global Constraints

- No new dependencies. `internal/...` never prints or exits.
- Godot 4 only for downloads; version regex `^4\.\d+(\.\d+)?(-(stable|rc\d+|beta\d+|dev\d+))?$`; plain number → `-stable` tag.
- Downloads: `https://github.com/godotengine/godot-builds/releases/download/<tag>/<asset>`; every asset verified against `SHA512-SUMS.txt` of the same tag. Base URL injectable for tests.
- `gd` never deletes files of linked editors; never changes `PATH`.
- Resolution never picks a `custom` build by version (only via `package.local.godot` or `default`).
- All installs (editors, templates) go through a staging dir inside the target parent and are moved into place only after every check passed.
- Editor names: managed `<version>[-dotnet]` (version as in `package.godot`, e.g. `4.7.2`, `4.8-beta1`); linked aliases `^[a-z0-9_-]+$`.
- Tests: stdlib `testing`; `cmd` tests not parallel; data dirs isolated via `isolateCache` (sets `HOME`, `XDG_*`, `APPDATA`, `LOCALAPPDATA`).
- Commit messages end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` and reference the issue.

## Review Focus

1. A zip whose symlink points outside the editor dir (`Godot.app/Contents/Frameworks/x -> ../../../../etc`) → install aborts, nothing in `editors/`. Test: Task 5 `TestExtractEditorRejectsSymlinkEscape`.
2. `gd editor install` for a version the platform has no asset for (e.g. .NET on an OS without that zip) → clear error listing found assets, no download of other files. Test: Task 4 `TestSelectEditorAssetMissing`.
3. `package.local.godot` names an editor that no longer exists → error naming the alias and `gd editor use --clear`. Test: Task 3 `TestResolveLocalNameMissing`.
4. `gd godot -- --headless --quit` returns Godot's non-zero exit code as gd's exit code. Test: Task 9 `TestGodotPassesExitCode`.
5. `gd templates install` when the target version dir exists → skipped without download; `--force` replaces it. Test: Task 11 `TestTemplatesInstallSkipsExisting`.

---

### Task 1: `[editor]` spec, `package.local.godot`, main scene (#43)

**Files:**
- Modify: `internal/godot/package.go`, `internal/godot/project.go`, tests
- Create: `internal/godot/local.go`, `internal/godot/local_test.go`

**Interfaces:**
- Produces:
  - `PackageEditorConfiguration{ Version string \`ini:"version"\`; Dotnet bool \`ini:"dotnet"\` }`; `NewPackage` validates `Editor.Version` when set → `ErrInvalidGodotVersion`.
  - `func ReleaseTag(version string) string` (`4.7.2` → `4.7.2-stable`, `4.8-beta1` unchanged).
  - `func MajorMinor(version string) string` (`4.7.2-stable` → `4.7`).
  - `const LocalFileName = "package.local.godot"`; `func LoadLocalEditor(dir string) (string, error)` (missing file → `""`); `func SaveLocalEditor(dir, name string) error` (empty name → remove file).
  - `func (p *Project) MainScene() (string, error)` reading `(?m)^run/main_scene="([^"]*)"`; missing → `ErrNoMainScene`.

- [ ] **Step 1: Write failing tests**: table `TestReleaseTag` (`4.7.2`, `4.7`, `4.8-beta1`, `4.7-rc2`, `4.8-dev7`); `TestNewPackageRejectsInvalidGodotVersion` (`3.6`, `4.x`, `4.7.2-alpha`); `TestEditorDotnetParsed`; `TestLocalEditorRoundTrip` (save, load, save "" removes file); `TestMainScene` (`res://main.tscn`, `uid://abc`, missing).
- [ ] **Step 2:** `go test ./internal/godot -v` → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `go test ./internal/godot -v` → PASS.
- [ ] **Step 5:** Commit `Godot version spec, package.local.godot and main scene` + `Refs #43`.

### Task 2: Data dirs, registry, `--version` parsing (#43)

**Files:**
- Create: `internal/editor/dirs.go`, `internal/editor/registry.go`, `internal/editor/version.go`, tests

**Interfaces:**
- Produces:
  - `func DataDir() (string, error)` — gd data dir per OS (spec); `func GodotDataDir() (string, error)` — Godot's data dir per OS.
  - `type Editor struct { Name, Path, Version string; Dotnet bool; Build string /* official|custom */; Source string /* managed|linked */ }`
  - `type Registry struct { Editors map[string]Editor; Default string }`; `func LoadRegistry(dataDir string) (*Registry, error)` (missing → empty); `func (r *Registry) Save(dataDir string) error`; file `editors.godot`, section `[aliases] default=…`, one section per editor.
  - `func ParseVersion(output string) (version string, dotnet bool, build string, err error)`: `4.7.2.stable.official.abc` → `4.7.2-stable, false, official`; `4.7.2.stable.mono.official.abc` → dotnet; `…custom_build…` → `custom`; `4.8.beta1.official.x` → `4.8-beta1`.
  - `func ProbeVersion(ctx context.Context, path string) (version string, dotnet bool, build string, err error)` runs `<path> --version` with a 10 s timeout.
- Test helper (new package): `internal/testutil/fakegodot` with `func Run() bool` (call first in `TestMain`; when `GD_FAKE_GODOT=1` acts as Godot: prints `GD_FAKE_GODOT_VERSION` for `--version`, appends args to `GD_FAKE_GODOT_LOG`, sleeps `GD_FAKE_GODOT_SLEEP`, exits with `GD_FAKE_GODOT_EXIT`) and `func Setup(t *testing.T, version string) (binary, log string)` (sets the env, returns `os.Args[0]`).

- [ ] **Step 1: Write failing tests**: `TestParseVersion` (table above + garbage → error); `TestProbeVersion` with fakegodot; `TestRegistryRoundTrip` (two editors + default); `TestDataDirs` (Linux via `XDG_DATA_HOME`, fallback `HOME`; skip other OS branches with `runtime.GOOS`).
- [ ] **Step 2:** `go test ./internal/editor -v` → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `go test ./internal/... -v` → PASS.
- [ ] **Step 5:** Commit `Editor registry, data dirs and version parsing` + `Refs #43`.

### Task 3: Resolution and warnings (#43)

**Files:**
- Create: `internal/editor/resolve.go`, test
- Modify: `cmd/search.go`, `cmd/install.go`, `cmd/helpers_test.go` (isolate data dirs), docs `docs/adr/0018-godot-version-spec.md`, `docs/adr/README.md`, `docs/configuration-files.md`

**Interfaces:**
- Produces:
  - `type Requirement struct { Version string /* release tag, "" = none */; Dotnet bool; Local string /* package.local.godot name */ }`
  - `func (r *Registry) Resolve(req Requirement) (Editor, error)` — rules 1–5 of the spec; errors `ErrNoEditor` (hint `gd editor install <version>`), `ErrUnknownEditor` (local name or default missing; hint `gd editor use --clear` / `gd editor default`), `ErrNoDefault`.
  - `func VersionWarning(projectVersion, editorVersion string) string` (non-empty when major.minor differ).
- `gd search` prefers `pkg.Editor.Version` (as major.minor) over `project.godot`; `gd install` and `gd search` print `VersionWarning` (stderr via `cmd.PrintErrln`) when `config/features` and `[editor] version` differ.

- [ ] **Step 1: Write failing tests**: `TestResolve` table for rules 1–5 (local wins incl. custom, default only when official+matching, managed before linked, linked by name order, custom never by version, dotnet flavour respected); `TestResolveLocalNameMissing`; `TestResolveNoRequirementUsesDefault`; `TestSearchPrefersEditorVersion` (cmd); `TestInstallWarnsOnVersionMismatch` (cmd, stderr contains `4.6` and `4.7`).
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3:** Implement; ADR 0018 (version spec, tag mapping, `package.local.godot`, warnings); config docs for `[editor]` and `package.local.godot`.
- [ ] **Step 4:** `go test ./...` → PASS.
- [ ] **Step 5:** Commit `Resolve the project's editor` + `Closes #43`.

### Task 4: godot-builds client (#57)

**Files:**
- Create: `internal/godotbuilds/godotbuilds.go`, test
- Modify: `internal/install/install.go` (hash-agnostic fetch)

**Interfaces:**
- Consumes: `godot.ReleaseTag`.
- Produces:
  - `const DefaultBaseURL = "https://github.com/godotengine/godot-builds/releases/download"`, `const DefaultAPIURL = "https://api.github.com/repos/godotengine/godot-builds/releases"`
  - `type Client struct { BaseURL, APIURL, Token string; HTTP *http.Client }`
  - `func (c Client) Sums(ctx context.Context, tag string) (map[string]string, error)` — asset → sha512; 404 → `ErrUnknownVersion`.
  - `func SelectEditorAsset(sums map[string]string, goos, goarch string, dotnet bool) (string, error)` — patterns of the spec in order, ignoring `debugsymbols`/`dSYM`; none → `ErrNoAsset` listing editor assets.
  - `func TemplatesAsset(tag string, dotnet bool) string`.
  - `func (c Client) Download(ctx context.Context, tag, asset, wantSHA512, dir string) (string, error)`.
  - In `internal/install`: `func FetchHash(ctx context.Context, client *http.Client, url, want, tmpDir string, newHash func() hash.Hash) (string, string, error)`; `Fetch` becomes a wrapper with `sha256.New` (callers unchanged).

- [ ] **Step 1: Write failing tests**: `TestSelectEditorAsset` table with the **real asset lists** of `4.7.2-stable` and `4.8-beta1` (copy from the spec context: universal vs arm64/x86_64 on macOS, all OS/arch, standard and .NET); `TestSelectEditorAssetMissing`; `TestSums` (parses `<hash>  <name>` lines, 404 → `ErrUnknownVersion`); `TestDownloadVerifiesSHA512` (mismatch → `install.ErrHashMismatch`).
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `go test ./internal/...` → PASS.
- [ ] **Step 5:** Commit `Add godot-builds client` + `Refs #57`.

### Task 5: Editor extraction and install (#57)

**Files:**
- Create: `internal/editor/install.go`, test

**Interfaces:**
- Consumes: `godotbuilds.Client`, `Registry`.
- Produces:
  - `func ExtractEditor(zipPath, target string) error` — keeps file modes and symlinks; rejects entries or symlink targets that resolve outside `target` (`ErrUnsafePath` from `internal/install`).
  - `func FindExecutable(dir, goos string) (gui, console string, err error)` — linux: the `Godot_v*` regular file; darwin: `*.app/Contents/MacOS/Godot` (console = gui); windows: `Godot_v*.exe` without `_console` (console = `…_console.exe` when present, else gui).
  - `func Install(ctx context.Context, c godotbuilds.Client, dataDir, version string, dotnet bool, goos, goarch string) (Editor, error)` — name `<version>[-dotnet]`; staging dir `editors/.staging-*`; rename to `editors/<name>`; returns the entry (caller registers it). Existing dir → `ErrAlreadyInstalled`.

- [ ] **Step 1: Write failing tests** (fake godot-builds server serving sums + zips built in the test): `TestInstallLinux` (binary executable), `TestInstallMacOSApp` (internal symlink preserved, executable path `…/Godot.app/Contents/MacOS/Godot`), `TestInstallWindows` (gui + console), `TestInstallDotnetFolder` (`GodotSharp/` kept), `TestExtractEditorRejectsSymlinkEscape`, `TestInstallHashMismatchLeavesNothing`.
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3:** Implement (pass `goos`/`goarch` explicitly so all layouts are testable on one OS).
- [ ] **Step 4:** `go test ./internal/...` → PASS.
- [ ] **Step 5:** Commit `Install Godot editors` + `Refs #57`.

### Task 6: `gd editor install | uninstall | list` (#57)

**Files:**
- Create: `cmd/editor.go`, `cmd/editor_test.go`
- Modify: `cmd/root.go` (hidden flags `--builds-url`, `--builds-api-url`), `cmd/helpers_test.go` (fake godot-builds server helper)

**Interfaces:**
- Produces: `func newEditorCmd(opts *options) *cobra.Command` with subcommands; `options` gains `buildsURL`, `buildsAPIURL` and `func (o *options) builds() godotbuilds.Client` (token from `GH_TOKEN` or `GITHUB_TOKEN`).
- `install [version] [--dotnet]`: default from `package.godot` (error outside project without version); `ErrAlreadyInstalled` → message, exit 0; registers; first installed editor becomes `default` when none is set.
- `uninstall <name>`: managed only (linked → error hint `unlink`); removes dir + entry; clears default.
- `list`: tabwriter `NAME VERSION DOTNET BUILD SOURCE PATH`, `*` default, `>` resolved project editor.

- [ ] **Step 1: Write failing tests**: install from `package.godot`, explicit version, `--dotnet`, already installed, uninstall managed, uninstall linked fails, list markers.
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `go test ./...` → PASS.
- [ ] **Step 5:** Commit `Add gd editor install, uninstall and list` + `Refs #57`.

### Task 7: `gd editor available` (#57)

**Files:**
- Modify: `internal/godotbuilds/godotbuilds.go`, `cmd/editor.go`, tests; Create `docs/adr/0017-editor-management.md`, index

**Interfaces:**
- Produces: `func (c Client) Available(ctx context.Context, pre bool) ([]string, error)` — GET `APIURL?per_page=100&page=N` until an empty page; `Authorization: Bearer <token>` when set; keeps tags matching the Godot 4 regex; stable only unless `pre`; sorted newest first (by release order of the API).
- `gd editor available [--pre]` prints one tag per line.

- [ ] **Step 1: Write failing tests**: paging (2 pages + empty), token header, `--pre` filter, Godot 3 tags dropped, API error status.
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3:** Implement; ADR 0017 (managed/linked, aliases, registry, resolution, custom builds, no PATH changes, godot-builds as source).
- [ ] **Step 4:** `go test ./...` → PASS.
- [ ] **Step 5:** Commit `Add gd editor available` + `Closes #57`.

### Task 8: `gd editor link | unlink | default | use` (#58)

**Files:**
- Modify: `cmd/editor.go`, `cmd/editor_test.go`

**Interfaces:**
- Consumes: `editor.ProbeVersion`, `Registry`, `godot.SaveLocalEditor`.
- `link <alias> <path>`: macOS `.app` → `Contents/MacOS/Godot`; alias regex, must not exist; probe; register `source=linked`.
- `unlink <alias>`: linked only; clears default.
- `default <name>`: must exist.
- `use <name>` / `use --clear`: inside a project; hint on stderr when `.gitignore` exists without `package.local.godot`.

- [ ] **Step 1: Write failing tests** (fakegodot): link official, link custom (`custom_build` → build=custom), invalid alias, duplicate alias, probe failure registers nothing, unlink managed fails, default unknown fails, use writes/clears local file, gitignore hint.
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `go test ./...` → PASS.
- [ ] **Step 5:** Commit `Add gd editor link, unlink, default and use` + `Closes #58`.

### Task 9: process start, `gd open`, `gd godot` (#59)

**Files:**
- Create: `internal/editor/run.go`, `internal/editor/detach_unix.go` (`//go:build !windows`), `internal/editor/detach_windows.go`, test; `cmd/open.go`, `cmd/godot.go`, tests

**Interfaces:**
- Produces:
  - `func Run(ctx context.Context, path string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error)` — exit code of the process, error only when it could not start.
  - `func StartDetached(path string, args []string) error` — new session/process group, stdio nil, `Process.Release`.
  - `cmd`: shared helper `resolveProjectEditor(cmd) (dir string, e editor.Editor, err error)` (loads package, local file, registry, prints `VersionWarning`).
- `gd editor open` and `gd open [--detach]`: args `-e --path <dir>`; foreground → return exit code as error `exitError{code}` so `Execute` exits with it.
- `gd godot -- <args>`: console executable (Windows), stdio passthrough, exit code.

- [ ] **Step 1: Write failing tests**: `TestOpenRunsEditorWithProject` (fakegodot log contains `-e --path <dir>`), `TestOpenShortcut`, `TestOpenDetachReturnsImmediately` (unix; fake sleeps 2 s, call returns < 1 s), `TestGodotPassesArgs`, `TestGodotPassesExitCode` (`GD_FAKE_GODOT_EXIT=3` → exit code 3), `TestOpenWithoutEditorFails` (hint `gd editor install`).
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3:** Implement; `Execute()` exits with the code of an `exitError`.
- [ ] **Step 4:** `go test ./...` → PASS.
- [ ] **Step 5:** Commit `Add gd open and gd godot` + `Closes #59`.

### Task 10: `gd start [scene]` (#60)

**Files:**
- Create: `cmd/start.go`, `cmd/start_test.go`

**Interfaces:**
- Consumes: `resolveProjectEditor`, `Project.MainScene`, `editor.Run`/`StartDetached`.
- No scene: `MainScene()` must exist (else `godot.ErrNoMainScene` with hint), args `--path <dir>`. With scene: relative → `res://<slash path>` and the file must exist; `res://`/`uid://` passed through; args `--path <dir> <scene>`.

- [ ] **Step 1: Write failing tests**: main scene, no main scene error, relative scene converted, missing relative scene error, `uid://` passthrough, `--detach`.
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `go test ./...` → PASS.
- [ ] **Step 5:** Commit `Add gd start` + `Closes #60`.

### Task 11: export templates (#45)

**Files:**
- Create: `internal/editor/templates.go`, test; `cmd/templates.go`, test; `docs/adr/0019-export-templates.md`, index

**Interfaces:**
- Produces:
  - `func TemplatesDir(godotDataDir string) string` → `<godot data>/export_templates`; `func SelfContainedTemplatesDir(editorPath string) (string, bool)` (`._sc_` or `_sc_` next to the binary; on macOS next to the `.app`) → `<dir>/editor_data/export_templates`.
  - `func InstallTemplates(ctx context.Context, c godotbuilds.Client, tag string, dotnet bool, templatesDir string, force bool) (string, error)` — download + verify `.tpz`, read `templates/version.txt`, extract `templates/*` to staging, move to `<templatesDir>/<version.txt>`; exists and `!force` → `ErrTemplatesInstalled`.
  - `func ListTemplates(templatesDir string) ([]string, error)`, `func UninstallTemplates(templatesDir, version string) error` (name must be a direct child; no path separators).
- `gd templates install [version] [--dotnet] [--force]`, `list`, `uninstall <version>`.

- [ ] **Step 1: Write failing tests**: install standard and mono (`4.7.2.stable`, `4.7.2.stable.mono`), `TestTemplatesInstallSkipsExisting` (no download; `--force` replaces), self-contained editor dir, list, uninstall rejects `../x`, hash mismatch leaves nothing.
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3:** Implement; ADR 0019.
- [ ] **Step 4:** `go test ./...` → PASS.
- [ ] **Step 5:** Commit `Add export template commands` + `Closes #45`.

### Task 12: README, docs, final checks (#18)

**Files:**
- Modify: `README.md`, `docs/configuration-files.md`

- [ ] **Step 1:** README: editor section (install, link, default, use, list, available), `gd open`, `gd start`, `gd godot`, templates; roadmap: editor rows `done`.
- [ ] **Step 2:** `gofmt -l .` empty, `go vet ./...`, `go test -race ./...` → PASS.
- [ ] **Step 3:** Smoke test on this machine: `gd editor available`, `gd editor install 4.7.2`, `gd editor list`, `gd templates install 4.7.2` into a scratch `XDG_DATA_HOME`/`HOME`, `gd godot -- --version`.
- [ ] **Step 4:** Commit `Document editor management` + `Closes #18`.
