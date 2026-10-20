# 0007. Install only `addons/<name>/**`

- Status: accepted
- Date: 2026-10-20
- Refs: gd#14

## Context

Asset downloads are repository archives: one top-level folder with
`addons/<name>/` plus demos, README and often a `project.godot`. The Godot editor
strips the top-level folder and lets the user pick files.

## Decision

`gd` strips the single top-level folder (if all entries share one that is not
`addons`) and installs only files below `addons/<name>/`. Files directly in
`addons/` are ignored. An archive without such files fails with "asset contains
no addons/ folder".

## Consequences

Projects are not polluted with demo files. Assets that are not addons (templates,
projects) cannot be installed. Executable bits are not preserved (deferred).
