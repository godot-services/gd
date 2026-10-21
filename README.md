# gd

[![Test](https://github.com/godot-services/gd/actions/workflows/test.yml/badge.svg)](https://github.com/godot-services/gd/actions/workflows/test.yml)

Package manager for Godot projects: installs addons from the official
[Godot Asset Library](https://godotengine.org/asset-library/) or a Git repository
into `addons/`, pinned and verified via `package-lock.godot`.

# Installation

```sh
go install github.com/godot-services/gd@latest
```

# Usage

Run `gd` in the folder that contains `project.godot`.

```sh
gd add 1709 --dev                                         # asset library ID
gd add https://github.com/foo/tool --version v1.2.0       # git (GitHub, GitLab, Codeberg)
gd install                                                # install everything from package-lock.godot
gd install --no-dev                                       # skip dev_assets (CI, exports)
gd remove tool

gd search dialogue                                        # find asset IDs (Godot version from project.godot or --godot)
gd outdated                                               # library assets with a newer version; exits 1 if any
gd update [name...]                                       # move assets to their latest version
gd cache info                                             # where the download cache is and how big
gd cache clean                                            # delete cached downloads
```

`gd add` writes `package.godot` and `package-lock.godot`; commit both. `gd install`
aborts when a download does not match the SHA-256 in `package-lock.godot`. Only
`addons/<name>/` folders of an asset are installed, and `gd` never overwrites
folders it did not install. See [configuration files](/docs/configuration-files.md)
and the [decision records](/docs/adr/README.md) for why it works this way.

## Editors

`gd` also manages Godot editors. Downloads come from
[github.com/godotengine/godot-builds](https://github.com/godotengine/godot-builds)
and are SHA-512 verified.

```sh
gd editor available [--pre] [--refresh]                   # versions you can install (cached for an hour)
gd editor install [version] [--dotnet]                    # no version: [editor] version from package.godot
gd editor list
gd editor link <alias> <path>                             # register an editor you installed yourself
gd editor unlink <alias>
gd editor default <name>                                  # global default editor
gd editor use <name>                                      # pick an editor on this machine (package.local.godot, not committed)
gd editor use --clear                                     # drop that pick again
gd editor uninstall <name>
gd editor shim install                                    # a godot command for your PATH that runs gd godot
gd editor shim uninstall
```

## Run

```sh
gd open [--detach]                                        # open the project in the editor (long form: gd editor open); outside a project: the project manager
gd start [scene] [--detach]                               # run the project (or a scene)
gd godot -- --version                                     # run the project's editor with any arguments
```

## Export templates

```sh
gd templates install [version] [--dotnet] [--force]
gd templates list
gd templates uninstall <version>
```

`gd editor shim install` writes `godot` (Windows: `godot.cmd`) into `<gd data>/bin`
and prints how to add that folder to your PATH; `gd` never changes PATH itself.
`<gd data>` is `~/.local/share/gd` (Linux, or `$XDG_DATA_HOME/gd`),
`~/Library/Application Support/gd` (macOS) or `%LOCALAPPDATA%\gd` (Windows).
Then `godot` starts the project's editor inside a project and the default editor
everywhere else.

See [ADR 0017](/docs/adr/0017-editor-management.md),
[0018](/docs/adr/0018-godot-version-spec.md),
[0019](/docs/adr/0019-export-templates.md),
[0020](/docs/adr/0020-release-list-cache.md) and
[0021](/docs/adr/0021-godot-shim.md).

> [!NOTE]
> Following roadmaps are currently `wip`

# configuration roadmap

| config file     | description                            | state                                    |
| --------------- | -------------------------------------- | ---------------------------------------- |
| `package.godot` | `gd` configuration for current project | [done](/docs/configuration-files.md) |
| `package-lock.godot` | resolved downloads and SHA-256 hashes | [done](/docs/configuration-files.md) |

# command roadmap

| cmd    | description                             | state |
| ------ | --------------------------------------- | ----- |
| asset  | manage assets of godot project          | todo  |
| editor | manage multiple version of godot editor | done  |

# command `asset` subcommands roadmap

| cmd           | description                             | state |
| ------------- | --------------------------------------- | ----- |
| asset install | install asset of current godot project  | done as `install` |
| asset add     | add asset to current godot project      | done as `add` |
| asset remove  | remove asset from current godot project | done as `remove` |

# command `editor` subcommands roadmap

| cmd              | description                                    | state |
| ---------------- | ---------------------------------------------- | ----- |
| editor list      | list current installed editor versions         | done  |
| editor install   | install godot editor for specified version     | done  |
| editor uninstall | uninstall godot editor for specified version   | done  |
| editor default   | set default godot editor for specified version | done  |
| editor shim      | `godot` command for your PATH that runs `gd godot` | done  |

# shortcut commands

| cmd     | description                                         | state |
| ------- | --------------------------------------------------- | ----- |
| add     | shortcut to add on asset to godot project           | done  |
| install | shortcut to install defined assets of godot project | done  |
| search   | search addons in the asset library                  | done  |
| outdated | show library assets with a newer version            | done  |
| update   | update assets to their latest version               | done  |
| cache    | inspect or clear the download cache                 | done  |
| open      | open the project in its editor                    | done  |
| start     | run the project or a scene                        | done  |
| godot     | run the project's editor with arguments           | done  |
| templates | install, list, uninstall export templates         | done  |
