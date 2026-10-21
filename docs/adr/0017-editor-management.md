# 0017. Editor management

- Status: amended by [0020](0020-release-list-cache.md)
- Date: 2026-10-20
- Refs: gd#18, gd#43, gd#57, gd#58; see [0018](0018-godot-version-spec.md)

## Context

Projects need a matching Godot editor, but editors come from many places: official
downloads, Steam, package managers, custom engine builds. All official builds
(stable, rc, beta, dev) are GitHub releases in `godotengine/godot-builds`; each
release has `SHA512-SUMS.txt` with every asset name and its SHA-512.

## Decision

- **godot-builds is the single source.** Downloads use the fixed URL
  `https://github.com/godotengine/godot-builds/releases/download/<tag>/<asset>`,
  and every asset is verified against `SHA512-SUMS.txt` of the same tag. The
  GitHub API is only used by `gd editor available` (`GH_TOKEN`/`GITHUB_TOKEN`
  raise its rate limit), so installing needs no API. Godot 4 only.
- **Managed vs linked.** Managed editors are installed by `gd` into
  `<data>/gd/editors/<name>`; `<name>` is the version plus `-dotnet` for .NET
  (`4.7.2`, `4.7.2-dotnet`, `4.8-beta1`). Linked editors are registered from a
  path under an alias (`^[a-z0-9_-]+$`); `gd` never deletes their files.
- **Registry.** `<data>/gd/editors.godot` (INI) holds one section per editor
  (`path`, `version`, `dotnet`, `build`, `source`) and `[aliases] default`.
  `build` is `official` or `custom`, read from `godot --version`.
- **Data dirs.** `$XDG_DATA_HOME/gd` or `~/.local/share/gd` (Linux),
  `~/Library/Application Support/gd` (macOS), `%LOCALAPPDATA%\gd` (Windows).
- **Resolution**, first match wins:
  1. `package.local.godot` `[editor] name`;
  2. `default` alias, when official and matching version and .NET flavour;
  3. managed official editor with that version and flavour;
  4. linked official editor with that version and flavour;
  5. error `no Godot <version> editor, run gd editor install`.

  Outside a project, or without `[editor] version`, only the `default` alias is used.
- **Custom builds** are only used through `package.local.godot` (`gd editor use`)
  or `default`, never by version matching.
- **No PATH changes, no shims.** `gd` runs the resolved editor itself.
- Installs go through a staging dir and are moved into place only after all checks passed.

## Consequences

Installing is deterministic, verified and independent of API limits. Editors from
anywhere can be used without gd owning them. A custom build can never be picked
by accident. Users who want `godot` on their PATH must arrange that themselves.
