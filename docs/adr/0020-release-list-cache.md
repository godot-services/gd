# 0020. Cache the godot-builds release list

- Status: accepted
- Date: 2026-10-21
- Refs: gd#65; amends [0016](0016-cache-commands.md) and [0017](0017-editor-management.md)

## Context

`gd editor available` asked the GitHub API on every run. Without a token the API
allows 60 requests per hour, and the release list spans several pages.

## Decision

- The list of all Godot 4 tags (stable and pre-releases) is cached in
  `<user cache dir>/gd/godot-builds-releases.json` together with the ETag of the
  first page and the time it was fetched. `--pre` only filters the output, so
  both views share one cache.
- A cache younger than one hour is used without any request.
- An older cache, or `--refresh`, sends one conditional request for the first
  page (`If-None-Match`). New releases always land on the first page, so a 304
  means the list is unchanged; a 304 does not count against the rate limit.
  A 200 fetches all pages again and replaces the cache.
- When the request fails and a cache exists, `gd` uses it and warns on stderr
  with its age. Without a cache the error is returned as before.
- An unreadable cache file is ignored; failing to write it does not fail the command.
- `gd cache info` and `gd cache clean` also cover this one file (amends 0016,
  which only covered `*.zip`). Other files in the cache dir are still never touched.

## Consequences

Repeated `gd editor available` runs are instant and rarely use the rate limit.
A release published within the last hour can be missing until `--refresh`.
Editing an old release on a later page is not noticed while the first page is
unchanged, which is fine because tags do not change.
