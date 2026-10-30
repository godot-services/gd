# 0003. INI format for `package.godot`

- Status: accepted
- Date: 2026-10-20
- Refs: [gd#4](https://github.com/godot-services/gd/issues/4)

## Context

Godot projects use an INI-like format (`project.godot`). Users already know it,
and `gopkg.in/ini.v1` is already a dependency.

## Decision

`package.godot` stays INI. Each asset is one key with an inline value
(`name={ library="1234" }`) or a nested section (`[assets.name]`). Exactly one
source key per asset: `library`, `git` (+ required `version`) or `registry`
(not supported yet).

## Consequences

No new dependency, familiar format. Inline tables are parsed by a small regexp
(only `key="value"` pairs), so unquoted values inside `{ }` are rejected.
