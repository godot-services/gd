# Architecture Decision Records

Every decision that shapes how `gd` behaves gets a short record here, so it can
be understood later: what the situation was, what was decided, and what it
costs. Records are never rewritten; a changed decision gets a new record that
supersedes the old one.

| #    | Decision                                                        | Status   |
| ---- | --------------------------------------------------------------- | -------- |
| [0001](0001-record-architecture-decisions.md) | Record architecture decisions          | accepted |
| [0002](0002-mvp-scope.md)                     | MVP scope: install addons only         | accepted |
| [0003](0003-ini-for-package-godot.md)         | INI format for `package.godot`         | accepted |
| [0004](0004-lockfile-with-sha256.md)          | Lockfile with SHA-256 per asset        | amended by 0013 |
| [0005](0005-lockfile-name.md)                 | Lockfile name `package-lock.godot`     | accepted |
| [0006](0006-git-sources-via-archive-urls.md)  | Git sources via archive URLs, https only | accepted |
| [0007](0007-install-only-addons.md)           | Install only `addons/<name>/**`        | accepted |
| [0008](0008-ownership-of-installed-folders.md) | gd only touches folders it installed  | amended by 0013 |
| [0009](0009-write-package-godot-with-ini-v1.md) | Write `package.godot` with gopkg.in/ini.v1 | accepted |
| [0010](0010-default-asset-names.md)           | Default asset names from titles        | amended by 0013 |
| [0011](0011-install-reconciliation.md)        | How `gd install` reconciles package and lock | accepted |
| [0012](0012-single-install-path.md)           | One install path for add and install   | amended by 0013 |
| [0013](0013-hardening-after-mvp-audit.md)     | Hardening after the MVP audit          | accepted |
| [0014](0014-zip-cache.md)                     | Cache downloaded zips by SHA-256       | accepted |
| [0015](0015-comfort-commands.md)              | Comfort commands: search, --no-dev, outdated, update | accepted |
| [0016](0016-cache-commands.md)                | Cache commands                         | accepted |
| [0017](0017-editor-management.md)            | Editor management: godot-builds, managed and linked editors | accepted |
| [0018](0018-godot-version-spec.md)            | Godot version spec and local editor choice | accepted |
| [0019](0019-export-templates.md)               | Export templates: source, verification, target dirs | accepted |

Template: copy [template.md](template.md).
