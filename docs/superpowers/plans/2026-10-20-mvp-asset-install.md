# MVP v0.1.0 Asset Installation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `gd add|install|remove` installs addons from the official Godot Asset Library or a Git host into `addons/`, reproducibly via `package.lock`.

**Architecture:** `internal/godot` owns the files (`package.godot`, `package.lock`, project dir), `internal/source` turns an asset reference into a zip URL, `internal/install` downloads, verifies and extracts `addons/**`. `cmd` wires them into cobra commands and is the only place that prints.

**Tech Stack:** Go (version from `go.mod`), `github.com/spf13/cobra`, `gopkg.in/ini.v1`, stdlib only otherwise (`net/http`, `archive/zip`, `crypto/sha256`, `net/http/httptest`).

**Spec:** `docs/superpowers/specs/2026-10-20-mvp-asset-install-design.md`

## Global Constraints

- No new dependencies.
- Library code (`internal/...`) never prints and never calls `os.Exit`; it returns errors.
- Official library base URL: `https://godotengine.org/asset-library/api`.
- Files: `package.godot`, `package.lock`, both INI, in the project dir (= current dir, must contain `project.godot`).
- Only `addons/**` from a zip is installed; the single top-level folder of the zip is stripped.
- `registry=` → error `registry sources are not supported yet`.
- Git `version` is required; supported hosts: `github.com`, `gitlab.com`, `codeberg.org`.
- Zip entry names use `/` (package `path`); disk paths use `filepath` (Windows).
- Tests use the stdlib `testing` package only (no testify), same package as the code (existing style).
- Every commit message ends with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`; reference the GitHub issue (`Refs #N` / `Closes #N`).

## Review Focus

1. Zip whose entries are **not** under one top-level folder (files at root, e.g. `addons/x/...` directly) → install still works, nothing stripped. Test: Task 9 `TestExtractWithoutTopLevelFolder`.
2. Download fails mid-way (HTTP 500 / connection drop) → no files in `addons/`, no lock change. Test: Task 9 `TestFetchHTTPError`, Task 10 `TestAddFailingDownloadLeavesProjectUntouched`.
3. Running `gd` in a directory without `project.godot` → clear error, no `package.godot` created. Test: Task 10 `TestAddOutsideProject`.
4. Git URL given without `--version`, or non-numeric library ID → validation error before any network call. Test: Task 5 `TestAssetValidate`.
5. Re-running `gd install` on an already installed project → idempotent, same files, no foreign-folder error for its own paths. Test: Task 11 `TestInstallTwiceIsIdempotent`.

---

## Milestone M0 — Foundation

### Task 1: Package parser returns errors (#5)

**Files:**
- Modify: `internal/godot/package.go`
- Modify: `internal/godot/package_test.go`

**Interfaces:**
- Produces: `NewPackage(source []byte) (*Package, error)` (unchanged signature, now returns errors); `parseCustomINIValue(value string) (PackageAssetConfiguration, error)` returns an error when no `key="value"` pair is found.

- [ ] **Step 1: Write failing tests**

```go
func TestProjectConfig(t *testing.T) {
	p, err := NewPackage([]byte(packageExample))
	if err != nil { t.Fatal(err) }
	if p.Package.Name != "My awesome package" || p.Editor.Version != "4.2.1" { t.Fatalf("got %+v", p) }
	if got := p.Assets["other_asset"].Git; got != "https://github.com/other/asset.git" { t.Fatal(got) }
	if got := p.DevAssets["custom_other_dev_asset"].Registry; got != "my-custom-asset-registry" { t.Fatal(got) }
	if len(p.Assets) != 3 || len(p.DevAssets) != 3 { t.Fatalf("assets %d dev %d", len(p.Assets), len(p.DevAssets)) }
}

func TestNewPackageInvalidSource(t *testing.T) {
	if _, err := NewPackage([]byte("[package\nname=x")); err == nil { t.Fatal("expected error") }
}

func TestNewPackageInvalidAssetValue(t *testing.T) {
	_, err := NewPackage([]byte("[assets]\nbroken=nonsense\n"))
	if err == nil || !strings.Contains(err.Error(), "broken") { t.Fatalf("expected error naming asset, got %v", err) }
}
```

- [ ] **Step 2:** `go test ./internal/godot -run 'TestProjectConfig|TestNewPackage' -v` → FAIL (process exits / no error).
- [ ] **Step 3:** In `package.go`: return `ini.LoadSources` and `cfg.MapTo` errors; `mapAssets` returns `(map, error)` and wraps with `fmt.Errorf("asset %q: %w", key.Name(), err)`; move the regexp to a package-level `var assetValueRe = regexp.MustCompile(...)`; drop all `fmt.Print*`, `os` import; rename `dev_assets` → `devAssets`.
- [ ] **Step 4:** `go test ./internal/godot -v` → PASS.
- [ ] **Step 5:** Commit `Return errors from package parser instead of exiting` + `Closes #5`.

### Task 2: Handle all os.Stat errors (#6)

**Files:**
- Modify: `internal/godot/editor.go`, `internal/godot/project.go`
- Delete: `internal/godot/proejct_test.go`
- Create: `internal/godot/project_test.go`

**Interfaces:**
- Produces: `NewProject(location string) (*Project, error)` — returns an error wrapping `os.ErrNotExist` when `project.godot` is missing; `ErrNoGodotProject = errors.New("no project.godot found")` wraps it: `fmt.Errorf("%w: %s", ErrNoGodotProject, dir)`.

- [ ] **Step 1: Write failing tests** in `project_test.go`

```go
func TestNewProjectInDirWithProjectFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ProjectFileName), nil, 0o644)
	p, err := NewProject(dir)
	if err != nil || p.Location != dir { t.Fatal(p, err) }
}
func TestNewProjectFromFileLocation(t *testing.T) { /* location = <dir>/project.godot → p.Location == dir */ }
func TestNewProjectWithoutProjectFile(t *testing.T) {
	_, err := NewProject(t.TempDir())
	if !errors.Is(err, ErrNoGodotProject) { t.Fatal(err) }
}
func TestNewProjectUnknownLocation(t *testing.T) { /* <tmp>/missing → errors.Is(err, os.ErrNotExist) */ }
```

- [ ] **Step 2:** `go test ./internal/godot -run TestNewProject -v` → `TestNewProjectWithoutProjectFile` FAILS (returns nil project and nil error today).
- [ ] **Step 3:** In `editor.go` and `locationDirectory`: `if err != nil { return …, err }` for every `os.Stat` error. `hasGodotProject` → `hasGodotProject(dir string) error` returning `ErrNoGodotProject`-wrapped error when missing, other stat errors as-is. Delete `proejct_test.go`.
- [ ] **Step 4:** `go test ./internal/godot -v` → PASS.
- [ ] **Step 5:** Commit `Handle all os.Stat errors in Editor and Project` + `Closes #6`.

### Task 3: Repo hygiene and CI (#7, #8, #9)

**Files:**
- Rename: `.github/CODEOWNER` → `.github/CODEOWNERS`
- Delete: `.github/test-coverage.yml`
- Modify: `.github/workflows/test.yml`, `cmd/root.go`

- [ ] **Step 1:** `git mv .github/CODEOWNER .github/CODEOWNERS`; `git rm .github/test-coverage.yml`.
- [ ] **Step 2:** `test.yml`: `actions/setup-go@v5` with `go-version-file: go.mod`; steps `go mod download`, `go vet ./...`, `go test ./... -cover -v`.
- [ ] **Step 3:** `cmd/root.go`: `Short: "Package manager for Godot projects"`, `Long:` two sentences (manages addons via `package.godot`/`package.lock`); remove generator comments and the `toggle` flag; set `SilenceUsage: true`.
- [ ] **Step 4:** `go vet ./... && go test ./... && go run . --help` → help shows the new text, no `--toggle`.
- [ ] **Step 5:** Commit `Clean up repo hygiene, CI and root command` + `Closes #7, closes #8, closes #9`.

---

## Milestone M1 — MVP v0.1.0

### Task 4: Spec and configuration docs (#10)

**Files:**
- Modify: `docs/configuration-files.md`

- [ ] **Step 1:** Replace the example with the spec's `package.godot` example (`library` / `git`+`version`), add a source-key table (copy from spec), a short `package.lock` section, and a note that `registry` is planned. Fix typo "differnt".
- [ ] **Step 2:** Commit `Update configuration docs to MVP source keys` + `Closes #10` (spec is already on this branch).

### Task 5: package.godot sources and validation (#11, part 1)

**Files:**
- Modify: `internal/godot/package.go`, `internal/godot/package_test.go`

**Interfaces:**
- Produces:
  - `PackageAssetConfiguration` gains `Library string`.
  - `func (a PackageAssetConfiguration) Validate() error` — errors: `ErrNoSource`, `ErrMultipleSources`, `ErrRegistryUnsupported` (`"registry sources are not supported yet"`), `ErrInvalidLibraryID` (not all digits), `ErrGitVersionRequired`.
  - `NewPackage` validates every asset and wraps errors as `asset %q: %w`.

- [ ] **Step 1: Write failing table test `TestAssetValidate`**

```go
cases := []struct{ a PackageAssetConfiguration; want error }{
	{PackageAssetConfiguration{Library: "1234"}, nil},
	{PackageAssetConfiguration{Git: "https://github.com/a/b", Version: "v1"}, nil},
	{PackageAssetConfiguration{}, ErrNoSource},
	{PackageAssetConfiguration{Library: "1", Git: "x", Version: "v"}, ErrMultipleSources},
	{PackageAssetConfiguration{Registry: "r", Version: "1"}, ErrRegistryUnsupported},
	{PackageAssetConfiguration{Library: "abc"}, ErrInvalidLibraryID},
	{PackageAssetConfiguration{Git: "https://github.com/a/b"}, ErrGitVersionRequired},
}
// for each: if !errors.Is(c.a.Validate(), c.want) → t.Errorf
```

  Also: `parseCustomINIValue` and `nestedSections` read the `library` key; `TestProjectConfig` now uses an example with `library`/`git` sources only (the old `registry` example fails validation — update `packageExample` to the spec example).

- [ ] **Step 2:** `go test ./internal/godot -run TestAssetValidate -v` → FAIL (undefined).
- [ ] **Step 3:** Implement as specified.
- [ ] **Step 4:** `go test ./internal/godot -v` → PASS.
- [ ] **Step 5:** Commit `Add library source and asset validation` + `Refs #11`.

### Task 6: package.godot load, write, add, remove (#11, part 2)

**Files:**
- Modify: `internal/godot/package.go`, `internal/godot/package_test.go`

**Interfaces:**
- Produces:
  - `const PackageFileName = "package.godot"`
  - `func LoadPackage(dir string) (*Package, error)` — missing file → empty `*Package` (non-nil maps), no error.
  - `func (p *Package) AddAsset(name string, a PackageAssetConfiguration, dev bool) error` — error `ErrAssetExists` if `name` is in `Assets` or `DevAssets`; validates `a`.
  - `func (p *Package) RemoveAsset(name string) error` — error `ErrAssetNotFound`.
  - `func (p *Package) Save(dir string) error`
  - `func (p *Package) AllAssets() map[string]PackageAssetConfiguration` — union of assets and dev assets.
- Approach: `Package` keeps the loaded `*ini.File` in an unexported field; `AddAsset` sets the key in `[assets]`/`[dev_assets]` with inline value `{ library="…" }` or `{ git="…", version="…" }`; `RemoveAsset` deletes the key or the nested `[assets.<name>]` section; `Save` uses `file.SaveTo`.

- [ ] **Step 1: Write failing tests**

```go
func TestPackageRoundTrip(t *testing.T) {
	dir := t.TempDir()
	src := "; my comment\n[package]\nname=\"Game\"\n\n[assets]\nfoo={ library=\"1\" }\n"
	os.WriteFile(filepath.Join(dir, PackageFileName), []byte(src), 0o644)
	p, _ := LoadPackage(dir)
	if err := p.AddAsset("bar", PackageAssetConfiguration{Git: "https://github.com/a/b", Version: "v1"}, true); err != nil { t.Fatal(err) }
	p.Save(dir)
	out, _ := os.ReadFile(filepath.Join(dir, PackageFileName))
	if !strings.Contains(string(out), "my comment") { t.Fatal("comment lost") }
	q, err := LoadPackage(dir)
	if err != nil || q.Assets["foo"].Library != "1" || q.DevAssets["bar"].Version != "v1" || q.Package.Name != "Game" { t.Fatal(q, err) }
}
func TestPackageRemoveNestedAsset(t *testing.T) { /* [assets.foo] section removed after RemoveAsset("foo") + Save + LoadPackage */ }
func TestPackageAddExisting(t *testing.T)       { /* errors.Is(err, ErrAssetExists) also across assets/dev_assets */ }
func TestLoadPackageMissingFile(t *testing.T)    { /* empty package, AddAsset works, Save creates the file */ }
```

- [ ] **Step 2:** `go test ./internal/godot -run 'TestPackage|TestLoadPackage' -v` → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `go test ./internal/godot -v` → PASS.
- [ ] **Step 5:** Commit `Load, write, add and remove assets in package.godot` + `Closes #11`.

### Task 7: package.lock (#12)

**Files:**
- Create: `internal/godot/lock.go`, `internal/godot/lock_test.go`

**Interfaces:**
- Produces:
  - `const LockFileName = "package.lock"`
  - `type LockEntry struct { Source, URL, SHA256, VersionString string; Paths []string }` (ini keys `source`, `url`, `sha256`, `version_string`, `paths` comma-separated)
  - `type Lock map[string]LockEntry`
  - `func LoadLock(dir string) (Lock, error)` — missing file → empty `Lock`.
  - `func (l Lock) Save(dir string) error` — sections sorted by name; empty lock → file removed if present.

- [ ] **Step 1:** `TestLockRoundTrip` (two entries, one with two paths → Save → LoadLock → `reflect.DeepEqual`), `TestLoadLockMissingFile` (empty, no error), `TestLockSaveEmptyRemovesFile`.
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3:** Implement with `ini.Empty()` / `ini.Load`.
- [ ] **Step 4:** `go test ./internal/godot -v` → PASS.
- [ ] **Step 5:** Commit `Add package.lock read and write` + `Closes #12`.

### Task 8: Sources (#13)

**Files:**
- Create: `internal/source/source.go`, `internal/source/source_test.go`

**Interfaces:**
- Consumes: `godot.PackageAssetConfiguration` (Task 5).
- Produces:
  - `const DefaultLibraryURL = "https://godotengine.org/asset-library/api"`
  - `type Resolved struct { URL, Title, VersionString, Source string }` — `Source` is `library:<id>` or `git:<repo>@<ref>`.
  - `type Library struct { BaseURL string; Client *http.Client }`
  - `func (l Library) Resolve(ctx context.Context, id string) (Resolved, error)` — `GET {BaseURL}/asset/{id}`; non-200 → error with status; empty `download_url` → `ErrNoDownloadURL`.
  - `func GitArchiveURL(repo, ref string) (string, error)` — `ErrUnsupportedGitHost`.
  - `func Resolve(ctx context.Context, lib Library, a godot.PackageAssetConfiguration) (Resolved, error)` — dispatches on `Library` / `Git`.
  - `func Slug(s string) string` — lowercase, runs of chars outside `[a-z0-9_]` → `_`, trimmed `_`.
  - `func RepoName(repo string) string` — last path segment without `.git`.

- [ ] **Step 1: Write failing tests**

```go
func TestGitArchiveURL(t *testing.T) {
	cases := []struct{ repo, ref, want string; err error }{
		{"https://github.com/a/b", "v1", "https://github.com/a/b/archive/v1.zip", nil},
		{"https://github.com/a/b.git", "v1", "https://github.com/a/b/archive/v1.zip", nil},
		{"https://gitlab.com/a/b/", "main", "https://gitlab.com/a/b/-/archive/main.zip", nil},
		{"https://codeberg.org/a/b", "abc123", "https://codeberg.org/a/b/archive/abc123.zip", nil},
		{"https://example.com/a/b", "v1", "", ErrUnsupportedGitHost},
	}
	// errors.Is + equality
}
func TestLibraryResolve(t *testing.T) {
	// httptest server: /asset/1234 → {"title":"Dialogic 2","version_string":"2.0","download_url":"https://x/y.zip"}
	// want Resolved{URL:"https://x/y.zip", Title:"Dialogic 2", VersionString:"2.0", Source:"library:1234"}
}
func TestLibraryResolveNotFound(t *testing.T)      { /* 404 → error */ }
func TestLibraryResolveNoDownloadURL(t *testing.T) { /* errors.Is(err, ErrNoDownloadURL) */ }
func TestSlug(t *testing.T) { /* "Dialogic 2" → "dialogic_2"; "  Foo--Bar! " → "foo_bar" */ }
```

- [ ] **Step 2:** `go test ./internal/source -v` → FAIL.
- [ ] **Step 3:** Implement; host check via `url.Parse(...).Host`.
- [ ] **Step 4:** `go test ./internal/source -v` → PASS.
- [ ] **Step 5:** Commit `Resolve assets from official library and git hosts` + `Closes #13`.

### Task 9: Installer (#14)

**Files:**
- Create: `internal/install/install.go`, `internal/install/install_test.go`

**Interfaces:**
- Produces:
  - `func Fetch(ctx context.Context, client *http.Client, url, wantSHA256, tmpDir string) (zipPath, gotSHA256 string, err error)` — streams to a file in `tmpDir` through `sha256`; non-200 → error; `wantSHA256 != "" && got != want` → `ErrHashMismatch`.
  - `func Extract(zipPath, projectDir string, owned []string) ([]string, error)` — returns installed top-level paths, sorted, slash form (`addons/dialogic`). Errors: `ErrNoAddons`, `ErrUnsafePath`, `ErrForeignPath`.
  - `func Remove(projectDir string, paths []string) error` — each path must start with `addons/` and stay inside `projectDir`, else `ErrUnsafePath`; missing paths are ignored.
- Algorithm for `Extract` (not determined by tests alone):
  1. Collect entry names; if all share the same first segment and that segment is not `addons`, strip it.
  2. Keep names starting with `addons/<x>/`; none → `ErrNoAddons`.
  3. Reject any kept name where `path.Clean` starts with `../` or is absolute → `ErrUnsafePath` (check all before writing anything).
  4. Extract into `os.MkdirTemp(projectDir, ".gd-staging-")`, `defer os.RemoveAll`.
  5. For each `addons/<x>`: if it exists in the project and is not in `owned` → `ErrForeignPath` (check all first). Then `os.RemoveAll` the old one and `os.Rename` staged → project.

- [ ] **Step 1: Write failing tests** (helper `makeZip(t, files map[string]string) string`)

```go
TestExtractStripsTopLevelAndKeepsAddonsOnly // repo-main/addons/x/a.gd, repo-main/demo/b.gd, repo-main/README.md → ["addons/x"], demo/ absent
TestExtractWithoutTopLevelFolder           // addons/x/a.gd at root → ["addons/x"]
TestExtractNoAddons                        // errors.Is(err, ErrNoAddons)
TestExtractZipSlip                         // repo/addons/../../evil.gd → ErrUnsafePath, nothing written in project
TestExtractForeignFolder                   // project has addons/x, owned=nil → ErrForeignPath, existing file unchanged
TestExtractReplacesOwnedFolder             // owned=["addons/x"] → old file gone, new file present
TestFetchVerifiesHash                      // httptest: wrong want → ErrHashMismatch; empty want → got == hex(sha256(body))
TestFetchHTTPError                         // 500 → error, no zip left in tmpDir
TestRemoveRejectsOutsideAddons             // Remove(dir, []string{"../x"}) → ErrUnsafePath
```

- [ ] **Step 2:** `go test ./internal/install -v` → FAIL.
- [ ] **Step 3:** Implement as specified.
- [ ] **Step 4:** `go test ./internal/install -v` → PASS.
- [ ] **Step 5:** Commit `Install addons from zip with hash verification` + `Closes #14`.

### Task 10: `gd add` (#15, part 1)

**Files:**
- Create: `cmd/assets.go` (shared wiring), `cmd/add.go`, `cmd/add_test.go`
- Modify: `cmd/root.go` (hidden persistent flag `--library-url`, default `source.DefaultLibraryURL`)

**Interfaces:**
- Consumes: Tasks 2, 5–9.
- Produces (in `cmd/assets.go`):
  - `func installAsset(ctx context.Context, lib source.Library, projectDir string, a godot.PackageAssetConfiguration, entry *godot.LockEntry) (godot.LockEntry, error)` — resolves when `entry == nil`, fetches with `entry.SHA256` when given, extracts with `entry.Paths` as `owned`, returns the new entry; temp dir via `os.MkdirTemp("", "gd-")`, removed afterwards.
  - `func newRootCmd() *cobra.Command` in `root.go` so tests get a fresh command tree; `Execute()` uses it.
- `gd add <ref>`: `ref` all digits → `Library`; starts with `http` → `Git` + required `--version`; name from `--name`, else `source.Slug(resolved.Title)` (library) or `source.RepoName` (git). Order: `NewProject(".")` → `LoadPackage` → validate/name conflict → `installAsset` → `AddAsset` → `Save` → lock `Save`. Prints `added <name> (<version_string or ref>) → addons/x`.

- [ ] **Step 1: Write failing tests** (helper `newTestProject(t)` = temp dir with `project.godot`, `t.Chdir`; helper `fakeLibrary(t, zip)` = httptest serving `/asset/{id}` and `/zip`)

```go
TestAddLibraryAsset                    // gd --library-url <srv> add 1234 → addons/x exists; package.godot has dialogic_2={ library="1234" }; lock sha256 non-empty, paths=addons/x
TestAddGitAssetRequiresVersion         // add https://github.com/a/b → error mentions --version, no files
TestAddDuplicateName                   // second add 1234 → ErrAssetExists
TestAddOutsideProject                  // no project.godot → errors.Is(err, godot.ErrNoGodotProject), no package.godot created
TestAddFailingDownloadLeavesProjectUntouched // zip endpoint 500 → no addons/, no package.godot, no package.lock
```

  A git happy path cannot run offline (hosts are fixed); Task 8 covers the URL logic and the rest is the shared `installAsset` path.
  `go.mod` is Go 1.23, so no `t.Chdir`: the helper does `os.Chdir` + `t.Cleanup` back, and `cmd` tests do not call `t.Parallel()`.

- [ ] **Step 2:** `go test ./cmd -run TestAdd -v` → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `go test ./cmd -v` → PASS.
- [ ] **Step 5:** Commit `Add gd add command` + `Refs #15`.

### Task 11: `gd install` and `gd remove` (#15, part 2)

**Files:**
- Create: `cmd/install.go`, `cmd/remove.go`, `cmd/install_test.go`, `cmd/remove_test.go`

**Interfaces:**
- Consumes: `installAsset` (Task 10), `install.Remove` (Task 9).
- `gd install`: for each `AllAssets()` sorted by name: `installAsset` with lock entry if present; save lock after each success. Then for lock entries without an asset: `install.Remove(entry.Paths)`, delete entry, save. Prints one line per asset.
- `gd remove <name>`: `ErrAssetNotFound` if not in package; `install.Remove(lock[name].Paths)`, `RemoveAsset`, delete lock entry, save both.

- [ ] **Step 1: Write failing tests**

```go
TestInstallFromLock          // after add: delete addons/ → install → files back, lock unchanged
TestInstallTwiceIsIdempotent // install, install → no error, same files
TestInstallHashMismatch      // tamper lock sha256 → errors.Is(err, install.ErrHashMismatch), addons/ unchanged
TestInstallResolvesMissingLockEntry // package.godot with library asset, no lock → lock entry written
TestInstallRemovesStaleLockEntry    // lock entry without asset → its paths deleted, entry gone
TestRemoveAsset              // add then remove → addons/x gone, package.godot and lock without entry
TestRemoveUnknown            // errors.Is(err, godot.ErrAssetNotFound)
```

- [ ] **Step 2:** `go test ./cmd -run 'TestInstall|TestRemove' -v` → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `go test ./... -v` → PASS.
- [ ] **Step 5:** Commit `Add gd install and gd remove commands` + `Closes #15`.

### Task 12: E2E, README, release (#16)

**Files:**
- Create: `cmd/e2e_test.go`
- Modify: `README.md`

- [ ] **Step 1:** `TestEndToEnd`: project A: `add 1234` → copy `package.godot` + `package.lock` to fresh project B → `install` in B → `addons/x` byte-identical to A → `remove dialogic_2` in B → clean.
- [ ] **Step 2:** `go test ./cmd -run TestEndToEnd -v` → PASS (behaviour already implemented; failure here means a real bug — fix it in the owning package).
- [ ] **Step 3:** README: install (`go install github.com/godot-services/gd@latest`), usage of `add`/`install`/`remove`, files section linking `docs/configuration-files.md`, roadmap table: `add`/`install`/`asset remove` → `done`.
- [ ] **Step 4:** Manual smoke test against the real library in a scratch Godot project: `gd add <a real addon id>` → `addons/` populated.
- [ ] **Step 5:** Commit `Add end-to-end test and README for v0.1.0` + `Closes #16`. Tag `v0.1.0` only after the branch is merged to `main`.
