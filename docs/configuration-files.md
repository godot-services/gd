# Configuration Files

`gd` uses two files in the root of a Godot project (next to `project.godot`):

- `package.godot`: the assets of the project, edited by you (or `gd add`/`gd remove`)
- `package-lock.godot`: the resolved download and its SHA-256 per asset, written by `gd` only

Both files use the INI format Godot uses for `project.godot`.

# `package.godot`

```ini
[package]
name="My game"
version="1.0.0"

[editor]
version="4.7.2"
dotnet=false

[assets]
dialogic={ library="1234" }
my_tool={ git="https://github.com/foo/tool", version="v1.2.0" }

[dev_assets]
gut={ library="1709" }

; alternative asset definition
[assets.other_tool]
git="https://gitlab.com/foo/other_tool"
version="main"
```

Every asset has exactly one source key:

| key        | meaning                                   | `version`                          |
| ---------- | ----------------------------------------- | ---------------------------------- |
| `library`  | asset ID in the official asset library    | informational, not used to resolve |
| `git`      | repository URL (GitHub, GitLab, Codeberg) | ref (tag, branch, commit); required |
| `registry` | custom registry                           | planned, not supported yet         |

`dev_assets` are installed like `assets` for now.

Asset names may only contain `a-z`, `0-9` and `_`.

`[editor]` pins the Godot editor of the project: `version` (`4.7.2`, `4.8-beta1`; a
plain number means the `-stable` release) and `dotnet` (default `false`). `gd search`
uses it as the Godot version, and `gd install` / `gd search` warn when its
major.minor differs from `config/features` in `project.godot`.

# `package.local.godot`

Personal, not committed (add it to `.gitignore`):

```ini
[editor]
name="my-build"
```

`name` is a registered editor (see `gd editor list`). It wins over version matching,
also for custom builds. Without it the editor is resolved by the `[editor]` version:
the `default` editor if it matches, else a managed, then a linked official editor.

# `package-lock.godot`

```ini
[dialogic]
source="library:1234"
url="https://github.com/coppolaemilio/dialogic/archive/master.zip"
sha256="9f86d0…"
version_string="2.0"
paths="addons/dialogic"
```

`gd install` downloads `url` and aborts if the SHA-256 differs, so every machine
gets the same files. `paths` lists the folders `gd` installed under `addons/`;
`gd remove` only ever deletes these.

Commit both files.

Downloaded zips are cached by SHA-256 in `<user cache dir>/gd/` (see [ADR 0014](adr/0014-zip-cache.md)); `gd cache info` shows it and `gd cache clean` clears it.
