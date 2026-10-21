# M4 Release Builds Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every `v*` tag produces a GitHub release with `gd` archives for 6 targets, `checksums.txt` and an attestation; `gd --version` prints the release version.

**Architecture:** `cmd/version.go` resolves the version (ldflags → BuildInfo → `dev`) into cobra's `Version`. GoReleaser (`.goreleaser.yaml`) builds and archives; `release.yml` runs it on tags and attests the archives; `test.yml` gets a snapshot job so PRs catch config and cross-compile errors.

**Tech Stack:** Go 1.23 (`go.mod`), cobra, GoReleaser v2.18.3, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-10-21-m4-release-builds-design.md`

## Global Constraints

- GoReleaser pinned to `v2.18.3` everywhere (workflows and local `go run github.com/goreleaser/goreleaser/v2@v2.18.3`).
- Actions: `actions/checkout@v4`, `actions/setup-go@v5` with `go-version-file: go.mod` (same as `test.yml`), `goreleaser/goreleaser-action@v7`, `actions/attest-build-provenance@v4`.
- ldflag: `-s -w -X github.com/godot-services/gd/cmd.buildVersion=v{{.Version}}`.
- Targets: goos `linux`, `darwin`, `windows` × goarch `amd64`, `arm64`; `CGO_ENABLED=0`, `-trimpath`.
- Archive name `gd_{{.Version}}_{{.Os}}_{{.Arch}}`, `tar.gz`, `zip` for windows, files `README.md`, `LICENSE`.
- `checksums.txt`, SHA-256; `release.mode: keep-existing`, `release.replace_existing_artifacts: true`, `changelog.use: github`.
- No new secrets; only `GITHUB_TOKEN`. Never push tags or create releases.
- ADRs 0020 (GoReleaser + release flow), 0021 (attestation, no notarization), 0022 (version string); check `git ls-tree origin/main docs/adr/` before committing and renumber on collision.
- Commits end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`; the last commit of the branch carries `Closes #41`, earlier ones `Refs #41`.

## Review Focus

- `gd add --version v1.2.0` must still set the git ref, not print gd's version (cobra's flag is root-only) — test in Task 1.
- `go install …@v0.4.0` without ldflags must print `v0.4.0`, not `dev` — covered by the BuildInfo case in Task 1.
- A local `go build` / `go run` must print `dev`, not `(devel)` — Task 1.
- A Windows target that no longer compiles must fail the PR, not the tag — snapshot job in Task 2.
- A release created by hand with notes must keep its notes — `keep-existing` in Task 2, checked in the config review (not testable before a tag).

---

### Task 1: `gd --version`

**Files:**
- Create: `cmd/version.go`, `cmd/version_test.go`, `docs/adr/0022-version-string.md`
- Modify: `cmd/root.go` (set `rootCmd.Version`), `docs/adr/README.md`

**Interfaces:**
- Produces: package var `buildVersion string` in `cmd` (target of the ldflag in Task 2); `func resolveVersion(ldflags string, info *debug.BuildInfo, ok bool) string`.

- [ ] **Step 1: Write the failing tests** in `cmd/version_test.go`

`TestResolveVersion`, table test:
- `resolveVersion("v0.4.0", &debug.BuildInfo{Main: debug.Module{Version: "v0.3.0"}}, true)` → `"v0.4.0"`
- `resolveVersion("", &debug.BuildInfo{Main: debug.Module{Version: "v0.3.0"}}, true)` → `"v0.3.0"`
- `resolveVersion("", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true)` → `"dev"`
- `resolveVersion("", &debug.BuildInfo{}, true)` → `"dev"`
- `resolveVersion("", nil, false)` → `"dev"`

`TestVersionFlag`: set `buildVersion = "v9.9.9"` (restore with `t.Cleanup`), run `newRootCmd()` with args `--version`, output captured via `SetOut` → equals `"gd version v9.9.9\n"`.

`TestAddVersionFlagIsNotGdVersion` (regression guard; passes already, the other two fail): `add, _, _ := newRootCmd().Find([]string{"add"})`; `add.Flags().Lookup("version").Usage` contains `git ref`.

- [ ] **Step 2: Run and see them fail**

Run: `go test ./cmd -run 'TestResolveVersion|TestVersionFlag|TestAddVersionFlag' -v`
Expected: build failure `undefined: resolveVersion` / `undefined: buildVersion`.

- [ ] **Step 3: Implement** `cmd/version.go` (`buildVersion` with a one-line comment that release builds set it via ldflags; `resolveVersion` per the spec order) and in `newRootCmd` set `Version: resolveVersion(buildVersion, debug.ReadBuildInfo())` — `ReadBuildInfo` returns `(info, ok)`, so call it first and pass both.

- [ ] **Step 4: Run tests**

Run: `go test ./... && go vet ./... && go run . --version`
Expected: all PASS; last line `gd version dev`.

- [ ] **Step 5: ADR 0022** "Version string in the binary" from `docs/adr/template.md`: ldflags in release builds, BuildInfo fallback for `go install`, `dev` otherwise; why not BuildInfo only (Go 1.23 does not stamp VCS tags for `go build`); `gd version vX` via cobra, no subcommand. Add index row.

- [ ] **Step 6: Commit** `Add gd --version` with `Refs #41`.

### Task 2: Release pipeline, docs

**Files:**
- Create: `.goreleaser.yaml`, `.github/workflows/release.yml`, `docs/adr/0020-release-builds-with-goreleaser.md`, `docs/adr/0021-build-attestation-instead-of-signing.md`
- Modify: `.github/workflows/test.yml`, `.gitignore` (create if missing; add `dist/`), `README.md` (Installation), `docs/adr/README.md`, spec status line `draft` → `accepted`

**Interfaces:**
- Consumes: `cmd.buildVersion` from Task 1.

- [ ] **Step 1: Write `.goreleaser.yaml`** with exactly the values in Global Constraints (`version: 2`, `project_name: gd`, one build `id: gd`, `main: .`, `binary: gd`).

- [ ] **Step 2: Validate and snapshot locally**

Run: `go run github.com/goreleaser/goreleaser/v2@v2.18.3 check` → expect `1 configuration file(s) validated`.
Run: `go run github.com/goreleaser/goreleaser/v2@v2.18.3 release --snapshot --clean`
Expected: `dist/` holds 5 `.tar.gz`, 1 `.zip` and `checksums.txt` with 6 lines; `cd dist && shasum -a 256 -c checksums.txt` all OK; `dist/gd_darwin_arm64*/gd --version` prints `gd version v<snapshot version>`; `tar tzf dist/gd_*_linux_amd64.tar.gz` lists `gd`, `README.md`, `LICENSE`.

- [ ] **Step 3: Write `release.yml`**: `on: push: tags: ['v*']`; job permissions `contents: write`, `id-token: write`, `attestations: write`; steps checkout (`fetch-depth: 0`), setup-go, `goreleaser/goreleaser-action@v7` with `distribution: goreleaser`, `version: v2.18.3`, `args: release --clean`, `env: GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`; then `actions/attest-build-provenance@v4` with `subject-path: |` `dist/*.tar.gz` / `dist/*.zip`.

- [ ] **Step 4: Add job `snapshot` to `test.yml`**: checkout (`fetch-depth: 0`), setup-go, goreleaser-action with the same version and `args: release --snapshot --clean`.

- [ ] **Step 5: Validate workflows** — if `actionlint` is available run it (`go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7`), expect no output; otherwise YAML-parse both files.

- [ ] **Step 6: ADRs 0020 and 0021** from the template, content from the spec's Decisions table and "Rejected" paragraph; index rows.

- [ ] **Step 7: README Installation**: first "Download" (releases page link `https://github.com/godot-services/gd/releases`, unpack, put `gd` on `PATH`; verify `sha256sum -c checksums.txt --ignore-missing` and `gh attestation verify gd_<version>_<os>_<arch>.tar.gz --repo godot-services/gd`; macOS: `xattr -d com.apple.quarantine gd` since the binary is not notarized), then `go install` as before. Mention `gd --version`.

- [ ] **Step 8: Final check**: `go vet ./... && go test ./...` PASS; `git status` shows no `dist/`.

- [ ] **Step 9: Commit** `Build release archives with GoReleaser` with `Closes #41`.
