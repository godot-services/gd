# 0018. Godot version spec and local editor choice

- Status: accepted
- Date: 2026-10-20
- Refs: [gd#43](https://github.com/godot-services/gd/issues/43)

## Context

Projects need to say which Godot editor they use, and a personal choice (a custom
build) must not end up in version control.

## Decision

- `package.godot` `[editor]` holds `version` (`4.7.2`, `4.8-beta1`) and `dotnet`.
  A plain number maps to the godot-builds tag `<version>-stable`; versions with a
  suffix are the tag. Only Godot 4 is supported.
- `package.local.godot` (not committed) holds `[editor] name`, the registered editor
  this checkout uses.
- Resolution order: local name, `default` alias (only when official and matching
  version and flavour), managed official editor, linked official editor (by name).
  Custom builds are never picked by version.
- `gd install` and `gd search` print a warning on stderr when the major.minor of
  `config/features` in `project.godot` and `[editor] version` differ. They never fail.
- `gd search` takes the Godot version from `--godot`, else `[editor] version`
  (as major.minor), else `project.godot`.

## Consequences

One committed line pins the editor for the whole team, while a local file allows
deviations. A wrong local name is an error with a hint instead of a silent fallback.
