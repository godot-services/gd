# 0005. Lockfile name `package-lock.godot`

- Status: accepted
- Date: 2026-10-20

## Context

The first draft used `package.lock` (like `Cargo.lock`, `composer.lock`). There
was no technical reason for that name. npm pairs `package.json` with
`package-lock.json`, keeping the extension of the manifest.

## Decision

The lockfile is `package-lock.godot`, next to `package.godot` and
`project.godot`, using the same `.godot` extension and INI format.

## Consequences

All `gd` files share one extension. Tools that recognise lockfiles by name
(e.g. GitHub's generated-file detection) do not know the name; add a
`.gitattributes` entry (`package-lock.godot linguist-generated`) if wanted.
