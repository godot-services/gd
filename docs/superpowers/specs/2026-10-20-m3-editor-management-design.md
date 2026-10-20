# M3 Editor Management — Design

Status: draft for review · Date: 2026-10-20 · Refs: gd#18, gd#43, gd#45 · Builds on: [ADRs](../../adr/README.md)

## Goal

`gd` manages Godot editors and export templates per project:

- install official Godot 4 editors (standard and .NET) and register editors
  installed elsewhere (Steam, package managers, custom builds);
- pick the right editor for a project from `package.godot`;
- open the project, run the game or run Godot with arguments through that editor;
- install the matching export templates.

## Context

- All Godot builds (stable, rc, beta, dev) are published as GitHub releases in
  `godotengine/godot-builds`, tag `<number>-<status>` (e.g. `4.7.2-stable`,
  `4.8-beta1`). `godotengine/godot` only has stable releases.
- Every release has `SHA512-SUMS.txt` listing **all asset names** with their
  SHA-512. Downloads use the fixed URL
  `https://github.com/godotengine/godot-builds/releases/download/<tag>/<asset>`,
  so installing needs no GitHub API (no rate limit).
- Editor asset names change between versions: 4.7.2 ships
  `_macos.universal.zip`, 4.8-beta1 ships `_macos.arm64.zip` and
  `_macos.x86_64.zip`. Godot 3 uses `x11`/`osx` names.
- Export templates: `Godot_v<tag>_export_templates.tpz` and
  `Godot_v<tag>_mono_export_templates.tpz`. A `.tpz` is a zip with a `templates/`
  folder containing `version.txt` (e.g. `4.7.2.stable`, `4.7.2.stable.mono`).
- Godot 4 looks for templates in `<godot data>/export_templates/<version.txt>/`:
  `~/.local/share/godot` (or `$XDG_DATA_HOME/godot`) on Linux,
  `~/Library/Application Support/Godot` on macOS, `%APPDATA%\Godot` on Windows.
  A self-contained editor (`._sc_` or `_sc_` next to the binary) uses
  `editor_data/export_templates/` next to the binary instead.
- `godot --version` prints e.g. `4.7.2.stable.official.abc123`,
  `4.7.2.stable.mono.official.abc123` or `4.7.2.stable.custom_build.abc123`.
- `project.godot` stores the main scene as `run/main_scene="res://…"` or
  `"uid://…"` in `[application]`.

## Non-goals

- Downloading Godot 3 editors or templates (Godot 3 editors can be linked).
- Custom export templates for custom engine builds (`gd templates link`, later issue).
- Version ranges in `package.godot`.
- Checking for an installed .NET SDK (later `gd doctor` check).
- Changing `PATH` or installing a `godot` shim.

## `package.godot`

```ini
[editor]
version="4.7.2"   ; Godot 4 only; "4.8-beta1", "4.7-rc2"; a plain number means stable
dotnet=true       ; optional, default false
```

- `version` maps to the release tag: `4.7.2` → `4.7.2-stable`, `4.8-beta1` → `4.8-beta1`.
- Validation: must match `^4\.\d+(\.\d+)?(-(stable|rc\d+|beta\d+|dev\d+))?$`.
- When `config/features` in `project.godot` has another major.minor than
  `[editor] version`, `gd install`, `gd search` and the editor commands print a warning.
- `gd search` uses `[editor] version` when set (before `project.godot`).

## `package.local.godot`

Per-machine choice, never committed:

```ini
[editor]
name="custombuild-x"
```

Written by `gd editor use <name>`, removed by `gd editor use --clear`. When the
project has a `.gitignore` without that entry, `use` prints a hint to add it.

## Editor registry

Data dir `<data>/gd`: `$XDG_DATA_HOME/gd` or `~/.local/share/gd` (Linux),
`~/Library/Application Support/gd` (macOS), `%LOCALAPPDATA%\gd` (Windows).

- `editors/<name>/`: managed editors, `<name>` = version, plus `-dotnet` for .NET
  (`4.7.2`, `4.7.2-dotnet`, `4.8-beta1`).
- `editors.godot` (INI):

  ```ini
  [aliases]
  default=4.7.2

  [4.7.2]
  path=/home/me/.local/share/gd/editors/4.7.2/Godot_v4.7.2-stable_linux.x86_64
  version=4.7.2-stable
  dotnet=false
  build=official
  source=managed
  ```

- `build` is `official` or `custom` (from `godot --version`); `source` is
  `managed` or `linked`. `gd` never deletes files of linked editors.

## Resolving the project's editor

First match wins:

1. `package.local.godot` `[editor] name` → that editor (warning when its version
   or .NET flavour differs from `package.godot`).
2. `default` alias, when it is `official` and matches version and .NET flavour.
3. A managed `official` editor with that version and flavour.
4. A linked `official` editor with that version and flavour (by name).
5. Error: `no Godot <version> editor, run gd editor install`.

Outside a project or without `[editor] version`: the `default` alias, else an
error with a hint to `gd editor default`. Custom builds are only used via
`use` or `default`, never by version matching.

## Commands

| command | behaviour |
| --- | --- |
| `gd editor available [--pre]` | lists versions from the GitHub API (`/repos/godotengine/godot-builds/releases`, pages of 100); default stable only; `--pre` adds rc/beta/dev; uses `GH_TOKEN` or `GITHUB_TOKEN` when set |
| `gd editor install [version] [--dotnet]` | default: version and flavour from `package.godot`; load `SHA512-SUMS.txt`, pick the asset, download, verify SHA-512, extract to a staging dir, move to `editors/<name>`, register; already installed → message, no download |
| `gd editor uninstall <name>` | managed only (linked → error with hint to `unlink`); deletes folder and entry, clears `default` if it pointed there |
| `gd editor link <alias> <path>` | binary or (macOS) `.app` (→ `Contents/MacOS/Godot`); runs `--version` (10 s timeout); stores version, flavour, build; alias `^[a-z0-9_-]+$`, must not exist yet |
| `gd editor unlink <alias>` | linked only; removes the entry, clears `default` if needed |
| `gd editor list` | `NAME VERSION DOTNET BUILD SOURCE PATH`; `*` marks `default`, `>` the project's editor |
| `gd editor default <name>` | sets the `default` alias |
| `gd editor use <name>` / `--clear` | writes / removes `package.local.godot` |
| `gd editor open [--detach]`, shortcut `gd open` | opens the project in its editor: `-e --path <project>` |
| `gd start [scene] [--detach]` | runs the game with the project's editor: `--path <project>` plus the scene; without scene the main scene from `project.godot`, none set → error |
| `gd godot -- <args>` | runs the project's editor with the given arguments, passes stdin/stdout/stderr through, returns its exit code |
| `gd templates install [version] [--dotnet] [--force]` | default: project version/flavour; download and verify the `.tpz`, target dir from `version.txt`; exists → skip unless `--force`; for a self-contained resolved editor its `editor_data/export_templates` |
| `gd templates list` | installed template versions in the Godot data dir |
| `gd templates uninstall <version>` | deletes that folder (e.g. `4.7.2.stable.mono`) |

**Foreground vs `--detach`:** without the flag Godot runs in the foreground with
output in the terminal and `gd` waits and returns Godot's exit code. With
`--detach` Godot starts as its own process without terminal output (like opening
it in Finder/Explorer) and `gd` returns at once: `setsid` on Linux/macOS,
`DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP` on Windows, stdio not connected.

**Scenes for `gd start`:** a path relative to the project (`levels/one.tscn`) is
turned into `res://levels/one.tscn` and must exist; `res://` and `uid://` are
passed through.

## Platforms

Asset patterns are tried in order; names containing `debugsymbols` or `dSYM` are ignored.

| OS / arch | standard | .NET |
| --- | --- | --- |
| linux/amd64 | `_linux.x86_64.zip` | `_mono_linux_x86_64.zip` |
| linux/arm64 | `_linux.arm64.zip` | `_mono_linux_arm64.zip` |
| darwin/* | `_macos.universal.zip`, then `_macos.<arm64\|x86_64>.zip` | `_mono_macos.universal.zip`, then `_mono_macos.<arm64\|x86_64>.zip` |
| windows/amd64 | `_win64.exe.zip` | `_mono_win64.zip` |
| windows/arm64 | `_windows_arm64.exe.zip` | `_mono_windows_arm64.zip` |

- Linux: the binary is made executable.
- macOS: `Godot.app` (or `Godot_mono.app`); executable `Contents/MacOS/Godot`.
- Windows: `open`/`start` use `Godot_….exe`; `gd godot` uses `…_console.exe`
  so output is visible.
- .NET zips contain a folder with the binary and `GodotSharp/`; the whole folder is kept.
- Editor extraction keeps executable bits and **symlinks inside the target**
  (needed for `.app` bundles); any entry or symlink pointing outside the target
  aborts the install. The strict addon extractor stays unchanged.

## Errors

- No `SHA512-SUMS.txt` for a tag (404) → `unknown Godot version <v>`, hint `gd editor available --pre`.
- No asset for the platform → error naming OS/arch and the editor assets found.
- SHA-512 mismatch → abort, nothing written.
- Install and templates go through a staging dir; an abort leaves nothing half-installed.
- `--version` failing on `link` → nothing registered.
- All errors are returned by `internal/...`; only `cmd` prints.

## Code structure

| package / file | responsibility |
| --- | --- |
| `internal/godotbuilds` | tag normalisation, `SHA512-SUMS.txt` parsing, asset selection, `Available` via API, download with SHA-512 |
| `internal/editor` | data dirs, registry (`editors.godot`), `--version` parsing, resolution, editor install/extract, templates dir and install, self-contained detection, process start (foreground/detached, build-tagged files for unix/windows) |
| `internal/godot` | `[editor] dotnet`, version validation, `package.local.godot`, main scene from `project.godot` |
| `cmd/editor.go`, `cmd/open.go`, `cmd/start.go`, `cmd/godot.go`, `cmd/templates.go` | commands |

## Testing

- Fake `godot-builds` server: `SHA512-SUMS.txt` and zips with the real layout per
  OS (Linux binary, macOS `.app` with an internal symlink, Windows exe pair, .NET folder).
- Fake Godot: the test binary re-executes itself (`os.Args[0]` + env var), prints a
  configurable `--version` and records its arguments to a file.
- Table tests: asset selection with the real names of 4.7.2 and 4.8-beta1 (standard
  and .NET, all OS/arch), tag normalisation, `--version` parsing (official, mono,
  custom_build), resolution rules 1–5, `version.txt` → template dir, data dirs per OS.
- Commands: install / uninstall / link / unlink / default / use / list / available
  (fake API with paging), open/start/godot argument building and exit codes,
  templates install/list/uninstall, staging cleanup on errors, symlink escape rejected.
- Detached start: process starts and `gd` returns without waiting (unix test; Windows by CI).

## Docs

- ADR 0017 Editor management (managed/linked, aliases, registry, resolution, custom builds, no PATH changes)
- ADR 0018 Godot version in `package.godot` and `package.local.godot`
- ADR 0019 Export templates (source, verification, target dirs, self-contained)
- README usage, roadmap tables; `docs/configuration-files.md` (`[editor]`, `package.local.godot`)

## GitHub

Split epic gd#18 into issues in M3:

1. Godot version in `package.godot`, `package.local.godot`, editor registry and resolution (#43)
2. `gd editor install | uninstall | list | available`
3. `gd editor link | unlink | default | use`
4. `gd editor open` / `gd open`, `gd godot`, `--detach`
5. `gd start [scene]`
6. Export templates `install | list | uninstall` (#45)

New issue without milestone: `gd templates link` for custom export templates.
