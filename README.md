# gd

[![Test](https://github.com/godot-services/gd/actions/workflows/test.yml/badge.svg)](https://github.com/godot-services/gd/actions/workflows/test.yml)

Package manager for Godot projects: installs addons from the official
[Godot Asset Library](https://godotengine.org/asset-library/) or a Git repository
into `addons/`, pinned and verified via `package.lock`.

# Installation

```sh
go install github.com/godot-services/gd@latest
```

# Usage

Run `gd` in the folder that contains `project.godot`.

```sh
gd add 1709 --dev                                         # asset library ID
gd add https://github.com/foo/tool --version v1.2.0       # git (GitHub, GitLab, Codeberg)
gd install                                                # install everything from package.lock
gd remove tool
```

`gd add` writes `package.godot` and `package.lock`; commit both. `gd install`
aborts when a download does not match the SHA-256 in `package.lock`. Only
`addons/<name>/` folders of an asset are installed, and `gd` never overwrites
folders it did not install. See [configuration files](/docs/configuration-files.md).

> [!NOTE]
> Following roadmaps are currently `wip`

# configuration roadmap

| config file     | description                            | state                                    |
| --------------- | -------------------------------------- | ---------------------------------------- |
| `package.godot` | `gd` configuration for current project | [done](/docs/configuration-files.md) |
| `package.lock`  | resolved downloads and SHA-256 hashes  | [done](/docs/configuration-files.md) |

# command roadmap

| cmd    | description                             | state |
| ------ | --------------------------------------- | ----- |
| asset  | manage assets of godot project          | todo  |
| editor | manage multiple version of godot editor | todo  |

# command `asset` subcommands roadmap

| cmd           | description                             | state |
| ------------- | --------------------------------------- | ----- |
| asset install | install asset of current godot project  | done as `install` |
| asset add     | add asset to current godot project      | done as `add` |
| asset remove  | remove asset from current godot project | done as `remove` |

# command `editor` subcommands roadmap

| cmd              | description                                    | state |
| ---------------- | ---------------------------------------------- | ----- |
| editor list      | list current installed editor versions         | todo  |
| editor install   | install godot editor for specified version     | todo  |
| editor uninstall | uninstall godot editor for specified version   | todo  |
| editor default   | set default godot editor for specified version | todo  |

# shortcut commands

| cmd     | description                                         | state |
| ------- | --------------------------------------------------- | ----- |
| add     | shortcut to add on asset to godot project           | done  |
| install | shortcut to install defined assets of godot project | done  |
