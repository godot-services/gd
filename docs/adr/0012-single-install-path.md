# 0012. One install path for add and install

- Status: accepted, amended by [0013](0013-hardening-after-mvp-audit.md)
- Date: 2026-10-20
- Refs: gd#15

## Context

`gd add` needs the resolved title for the default name and the name-conflict
check before installing; `gd install` resolves only assets without a lock entry.

## Decision

`gd add` resolves the source first, checks the name, then calls the same
`installAsset` function as `gd install`, passing a lock entry with URL but no
hash. `installAsset` resolves only when it gets no URL. `add` saves
`package.godot` and then the lockfile after a successful install.

## Consequences

One code path for download, verification and extraction. If the process dies
between the two saves, the asset's folder has no owner and must be deleted by
hand (deferred: save the lock first).
