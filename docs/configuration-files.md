# Configuration Files

`gd` uses two files in the root of a Godot project (next to `project.godot`):

- `package.godot`: the assets of the project, edited by you (or `gd add`/`gd remove`)
- `package.lock`: the resolved download and its SHA-256 per asset, written by `gd` only

Both files use the INI format Godot uses for `project.godot`.

# `package.godot`

```ini
[package]
name="My game"
version="1.0.0"

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

# `package.lock`

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
