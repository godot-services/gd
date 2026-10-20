# 0002. MVP scope: install addons only

- Status: accepted
- Date: 2026-10-20
- Refs: milestone "M1 MVP v0.1.0", spec `docs/superpowers/specs/2026-10-20-mvp-asset-install-design.md`

## Context

`gd` is planned as asset manager and editor version manager. `gdar` (custom
registry) and the OpenAPI spec are empty scaffolds. The fastest useful result is
installing assets that already exist in the official asset library.

## Decision

v0.1.0 contains `gd add`, `gd install`, `gd remove` for two sources: the official
asset library (by ID) and Git repositories. Out of scope: custom registries
(`registry=` returns an error), `search`, `update`, `outdated`, separate handling
of `dev_assets` (parsed and installed like `assets`), editor management.
`gd` works in the current directory, which must contain `project.godot`.

## Consequences

Usable without running `gdar`. `registry=` support follows the read-only `gdar`
milestone.
