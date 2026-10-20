# 0019. Export templates

- Status: accepted
- Date: 2026-10-20
- Refs: gd#45

## Context

Exporting a project needs export templates matching the editor version. Godot looks
for them in its own data directory, or next to a self-contained editor.

## Decision

- Source: `Godot_v<tag>_export_templates.tpz` (or `_mono_`) from godot-builds, verified
  against the `SHA512-SUMS.txt` of the same tag like every other download.
- A `.tpz` is a zip with a `templates/` folder holding `version.txt`. It is unpacked
  with the hardened editor extractor into a `.staging-*` dir inside the templates dir,
  and `templates/` is moved to `<templates dir>/<version.txt>` only after all checks
  passed. `version.txt` must be a plain name (letters, digits, dots, no separators).
  Staging is removed on every error path.
- Target: `<godot data>/export_templates/`. When the project's resolved editor is
  self-contained (`._sc_` or `_sc_` next to the binary, next to the `.app` on macOS),
  `editor_data/export_templates/` next to it. Resolution errors fall back to the
  Godot data dir.
- Existing templates are skipped without downloading (exit 0, "already installed");
  `--force` replaces them.
- `gd templates list` shows the Godot data dir; `uninstall <version>` only removes a
  direct child of the templates dir.

## Consequences

Re-running install is cheap and safe. `list` and `uninstall` do not look at
self-contained editors' template folders.
