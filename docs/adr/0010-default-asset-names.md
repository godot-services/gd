# 0010. Default asset names from titles

- Status: accepted, amended by [0013](0013-hardening-after-mvp-audit.md)
- Date: 2026-10-20
- Refs: [gd#15](https://github.com/godot-services/gd/issues/15)

## Context

`gd add <id|url>` needs a key for `package.godot` when `--name` is not given.

## Decision

The default name is the slug of the resolved title: lowercase, runs of characters
outside `[a-z0-9_]` become `_`. For library assets the title comes from the
library ("GUT - Godot Unit Testing (Godot 4)" → `gut_godot_unit_testing_godot_4`),
for Git assets it is the repository name (`my-tool` → `my_tool`). An existing
name is an error.

## Consequences

Predictable, valid INI keys; long library titles give long names, `--name`
overrides. `--name` itself is not validated yet (deferred; e.g. `DEFAULT`
collides with the INI default section).
