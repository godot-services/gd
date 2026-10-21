# 0016. Cache commands

- Status: amended by [0020](0020-release-list-cache.md)
- Date: 2026-10-20
- Refs: gd#49, builds on [0014](0014-zip-cache.md)

## Context

The zip cache from ADR 0014 has no size limit and no eviction. Users had to find
and delete the folder by hand.

## Decision

- `gd cache info` prints the cache folder, the number of cached zips and their size.
- `gd cache clean` deletes all cached zips.
- Both only look at `*.zip` files directly in `<user cache dir>/gd`; other files
  and folders are never touched.
- Subcommands follow the noun-verb pattern (`gd cache clean`), not `gd clear cache`.
- No `--unused` option yet (keep zips referenced by the current lockfile): the
  next `gd install` downloads what is missing anyway.

## Consequences

Clearing the cache is one command. After `gd cache clean` the next `gd install`
downloads every asset once again.
