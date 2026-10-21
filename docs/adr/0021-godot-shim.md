# 0021. godot shim and the editor outside a project

- Status: accepted
- Date: 2026-10-21
- Refs: gd#64; supersedes the "No PATH changes, no shims" point of [0017](0017-editor-management.md)

## Context

ADR 0017 ruled out shims: `gd` runs the resolved editor itself, and users who
want `godot` on their PATH arrange that themselves. In practice people, scripts
and IDEs call `godot` directly, and `gd godot --` already resolves the right
editor inside and outside a project. `gd open` failed outside a project.

## Decision

- `gd editor shim install` writes `<gd data>/bin/godot` (unix, mode 0755) or
  `<gd data>/bin/godot.cmd` (Windows), overwriting an existing one. It prints how
  to add that folder to PATH when it is not there yet.
- **gd never changes PATH or shell profiles itself**; installing the shim is opt-in.
- The shim runs `gd godot -- <args>` with the absolute path of the `gd` that
  installed it. When the `gd` on PATH is that same binary, its PATH entry is
  used, so a package manager's `bin/gd` link survives upgrades (`os.Executable`
  resolves links on Linux). When that file is gone, it falls back to `gd` from PATH.
  The absolute path keeps the shim working where PATH lacks `gd`, for example
  in IDEs and GUI apps; the fallback keeps it working after `gd` moved or when
  it was installed from a temporary `go run` build.
- The Windows shim uses `goto` instead of `( )` blocks, because paths such as
  `Program Files (x86)` would end a block, and doubles `%` in the path.
- `gd editor shim uninstall` removes the file.
- `gd editor link` refuses the shim itself: as a linked editor it would start
  `gd godot` again and again.
- `gd open` outside a project starts the `default` editor with
  `--project-manager` instead of failing. `gd start` still needs a project.

## Consequences

`godot` on PATH always starts the editor `gd` would pick: the project's editor
inside a project, the default editor elsewhere. Every `godot` call costs one
extra process start. A shim written by an old `gd` keeps calling that binary
while it exists; `gd editor shim install` refreshes it. On Windows the `.cmd`
shim brings cmd's own limits: Ctrl+C asks "Terminate batch job (Y/N)?" and
arguments go through cmd's parsing.
