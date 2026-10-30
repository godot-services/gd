# 0011. How `gd install` reconciles package and lock

- Status: accepted
- Date: 2026-10-20
- Refs: [gd#15](https://github.com/godot-services/gd/issues/15)

## Context

`package.godot` is edited by hand: assets get added, removed, renamed or their
source changes. The lockfile must follow without losing track of installed
folders.

## Decision

`gd install` runs in this order:

1. Lock entries without an asset in `package.godot` are removed together with
   their folders — first, so a renamed asset does not see its own old folder as
   foreign.
2. For every asset (sorted by name): if the lock entry's source no longer matches
   `package.godot` (other library ID, git URL or ref), the asset is resolved again
   while keeping ownership of its folders; otherwise the locked URL is downloaded
   and verified.
3. Folders an asset no longer ships are removed after a successful install.
4. The lockfile is saved after every asset, so a later failure keeps an accurate
   record of what is installed.

## Consequences

Editing `package.godot` and running `gd install` does what users expect. A hash
mismatch in a later asset leaves earlier assets updated (the lock stays truthful),
which deviates from the spec's "abort, no changes" for the whole run.
