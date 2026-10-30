# 0014. Cache downloaded zips by SHA-256

- Status: accepted
- Date: 2026-10-20
- Refs: [gd#30](https://github.com/godot-services/gd/issues/30)

## Context

`gd install` downloaded every asset on every run, even when nothing changed.
With many addons every CI run cost one download per addon.

## Decision

Downloads are stored as `<user cache dir>/gd/<sha256>.zip`
(`os.UserCacheDir()`: `~/Library/Caches` on macOS, `$XDG_CACHE_HOME` or
`~/.cache` on Linux, `%LocalAppData%` on Windows).

- When the lock entry has a hash and the cached file exists, `gd` hashes the
  cached file again and uses it only if it matches; otherwise it downloads again.
- New downloads land directly in the cache folder and are renamed to their hash,
  so no copy across file systems is needed.
- Without a usable cache dir `gd` downloads into a temp dir as before.
- There is no size limit and no eviction. Delete the folder to clear the cache.

## Consequences

Repeated `gd install` runs and CI with a cached folder need no network for
unchanged assets. A cached zip still installs when the upstream archive changed
later, which is intended: the lockfile pins the exact bytes. The cache grows with
every new asset version.
