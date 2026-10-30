# 0013. Hardening after the MVP audit

- Status: accepted
- Date: 2026-10-20
- Refs: [gd#20](https://github.com/godot-services/gd/issues/20), [gd#21](https://github.com/godot-services/gd/issues/21), [gd#22](https://github.com/godot-services/gd/issues/22), [gd#23](https://github.com/godot-services/gd/issues/23), [gd#24](https://github.com/godot-services/gd/issues/24), [gd#25](https://github.com/godot-services/gd/issues/25)
- Amends: [0004](0004-lockfile-with-sha256.md), [0008](0008-ownership-of-installed-folders.md), [0010](0010-default-asset-names.md), [0012](0012-single-install-path.md)

## Context

An audit of the merged MVP found cases where `gd` installed unverified files,
lost track of installed folders, or hung forever. Each could leave a project in
a state the user can only fix by hand.

## Decision

- **No hash, no install (amends 0004):** `gd install` refuses a lock entry that
  has a `url` but no `sha256`, instead of installing it unverified.
- **Lockfile first (amends 0012):** `gd add` saves `package-lock.godot` before
  `package.godot`. If the second save fails, the lock entry is stale and the next
  `gd install` removes it with its folders, so nothing is left without an owner.
- **Asset names (amends 0010):** `--name` must only contain `a-z`, `0-9` and `_`,
  the same characters as default names. Other names could clash with INI syntax,
  e.g. `DEFAULT` is the INI default section and would drop the lock entry.
- **Folder names (amends 0008):** addon folders with a `,` are rejected, because
  `paths` in the lockfile is comma-separated. Folders with `..` inside a name
  (`addons/my..tool`) are allowed; only `..` as a path segment is unsafe.
- **Timeout:** all HTTP requests (library lookups and downloads) time out after
  10 minutes.

## Consequences

`gd` fails loudly instead of leaving hidden damage. Assets with a comma in an
addon folder name cannot be installed. A download that takes longer than 10
minutes fails.
