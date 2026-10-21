# M4 Package Managers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every stable `v*` tag updates `Casks/gd.rb` in `godot-services/homebrew-tap` and `bucket/gd.json` in `godot-services/scoop-bucket`.

**Architecture:** Two new sections in `.goreleaser.yaml` (`homebrew_casks`, `scoops`) plus `release.prerelease: auto`; `release.yml` passes `TAP_GITHUB_TOKEN`. No Go code changes, so there is no unit test; the snapshot output is the test.

**Tech Stack:** GoReleaser v2.18.3, GitHub Actions, Homebrew 7 (local `brew style`).

**Spec:** `docs/superpowers/specs/2026-10-21-m4-package-managers-design.md`

## Global Constraints

- The YAML in the spec's `.goreleaser.yaml` section is used verbatim, plus `prerelease: auto` under `release:`.
- Token only as `{{ .Env.TAP_GITHUB_TOKEN }}` in the config and `secrets.TAP_GITHUB_TOKEN` in `release.yml`; not in `test.yml`.
- Tap and bucket repos are already seeded (README, LICENSE on `main`); this plan pushes nothing outside `gd`.
- ADR 0023 (check `git ls-tree origin/main docs/adr/` for collisions first).
- Commits end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`; the last one carries `Closes #42`.

## Review Focus

- A snapshot or PR build without `TAP_GITHUB_TOKEN` must still succeed (missing env in a template must not fail it) — Task 1 Step 2.
- macOS users installing via brew must be able to run the unnotarized binary — hook present in the generated cask, Task 1 Step 3.
- Linux Homebrew users must get the linux archives — `on_linux` section, Task 1 Step 3.
- A prerelease tag must not reach brew/scoop users — `skip_upload: auto` in the generated config, Task 1 Step 1 review.
- Windows arm64 users must get the arm64 zip from Scoop — `architecture.arm64` in the manifest, Task 1 Step 3.

---

### Task 1: Homebrew and Scoop publishing

**Files:**
- Modify: `.goreleaser.yaml`, `.github/workflows/release.yml`, `README.md` (Installation), `docs/adr/README.md`
- Create: `docs/adr/0023-package-managers.md`

- [ ] **Step 1: Add** `homebrew_casks`, `scoops` and `release.prerelease: auto` to `.goreleaser.yaml` as in the spec; add `TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}` to the GoReleaser step's `env` in `release.yml`.

- [ ] **Step 2: Validate**

Run: `go run github.com/goreleaser/goreleaser/v2@v2.18.3 check` → `1 configuration file(s) validated`, no deprecation warning.
Run (with `TAP_GITHUB_TOKEN` unset): `go run github.com/goreleaser/goreleaser/v2@v2.18.3 release --snapshot --clean` → exit 0.
Run: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7` → no output.

- [ ] **Step 3: Inspect the generated files**

`dist/homebrew/Casks/gd.rb`: `on_macos` and `on_linux` blocks, each with `on_intel`/`on_arm` (or equivalent) URLs `https://github.com/godot-services/gd/releases/download/v<version>/gd_<version>_<os>_<arch>.tar.gz`, `sha256` values equal to the lines in `dist/checksums.txt`, `binary "gd"`, and the `xattr` postflight hook.
`dist/scoop/gd.json`: `architecture.64bit` and `architecture.arm64` with the windows zip URLs and hashes from `checksums.txt`, `bin: gd.exe`.
Run: `brew style dist/homebrew/Casks/gd.rb` → `no offenses detected` (if brew refuses files outside a tap, copy into a temp tap via `brew tap-new --no-git godot-services/test`, run, then `brew untap godot-services/test`).

- [ ] **Step 4: ADR 0023** from the template, content from the spec's Decisions table and "Rejected" paragraph; index row.

- [ ] **Step 5: README Installation**: Homebrew (`brew install godot-services/tap/gd`) and Scoop (`scoop bucket add godot-services https://github.com/godot-services/scoop-bucket`, `scoop install gd`) first, then the existing download and `go install` parts.

- [ ] **Step 6: Final check**: `go vet ./... && go test ./...` PASS; `git status` shows no `dist/`.

- [ ] **Step 7: Commit** `Publish gd to Homebrew and Scoop` with `Closes #42`.
