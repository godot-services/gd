# 0009. Write `package.godot` with gopkg.in/ini.v1

- Status: accepted
- Date: 2026-10-20
- Refs: gd#11

## Context

`gd add` and `gd remove` must change `package.godot` without losing the user's
content. A custom writer would preserve formatting exactly but is new code to
maintain.

## Decision

`package.godot` is loaded and saved with `gopkg.in/ini.v1`:

- Comments survive. Values are written unquoted and keys aligned
  (`name = Game`); empty `[assets]` / `[dev_assets]` sections may be added.
  Inline comments move to their own line above the key.
- `SpaceBeforeInlineComment`: only ` ;` / ` #` start a comment, so values such as
  `name="A;B"` or `version="rel#1"` survive.
- Duplicate sections (e.g. two `[assets]` after a merge) are merged instead of
  ignoring the second one, which previously made `gd install` delete addons.
- Nested sections are read with `KeysHash`, because `Key()` creates missing keys
  that would be written back empty.

## Consequences

`package.godot` is reformatted on the first `gd add`/`gd remove`; the content
stays the same. Revisit with a custom writer if users object to the reformatting.
