# 0024. Read and write project.godot with an own minimal parser

- Status: accepted
- Date: 2026-10-21
- Refs: gd#46, docs/superpowers/specs/2026-10-21-m5-project-settings-design.md, ADR 0015

## Context

`project.godot` looks like INI, but values are Godot variant text: strings, numbers,
`PackedStringArray(...)`, `Vector2i(...)`, arrays, multi-line dictionaries and
`Object(...)`. `gopkg.in/ini.v1` cannot round-trip them (ADR 0015), so `gd` only read
single lines with regular expressions. Letting the editor rewrite the file would lose
the layout of everything we did not mean to touch, and needs an installed editor.

## Decision

- `internal/godot/settings.go` splits the file into blocks (preamble, section header,
  entry) and keeps each block's original text. Load + save without changes is byte for
  byte identical.
- An entry's value ends when it is balanced: outside strings every `(`, `[`, `{` is
  closed. Strings honour `\` escapes and may span lines. A value still open at the end
  of the file is the error `project.godot:<line>: unterminated value`.
- A setting path maps like Godot does: the part before the first `/` is the section,
  the rest is the key. A path without `/` lives before the first section.
- A key in quotes (Godot quotes keys with spaces, non-ASCII bytes or `=";[]`, e.g.
  `"Move Left"`) is read under its decoded name; a new key that needs quotes is
  written quoted the same way.
- `set` replaces an existing value in place. A new key goes to the end of its section;
  a new section is inserted alphabetically with a blank line before it. New lines use
  the file's line ending (CRLF if the first line has it). `unset` removes the entry;
  an empty section stays.
- The file is written atomically (temp file in the same dir, rename), keeping its mode.
- Type inference for `gd config set`: (1) if the current value is a string, the new
  value is a string, unless it starts with `"`; (2) otherwise `true`, `false`, `null`,
  numbers and literals starting with `"`, `&"`, `^"`, `[`, `{` or `Name(` are raw;
  (3) anything else is an escaped, quoted string; (4) an unbalanced raw value is
  rejected and the file stays unchanged.
- Only strings and `PackedStringArray` are decoded. Everything else stays raw text.
- `Project.GodotVersion` and `Project.MainScene` read through this parser. It replaces
  the single-line regular expressions from ADR 0015.

## Consequences

Edits are byte-preserving and need no editor. Values outside the inference rules must
be written in Godot syntax. Setting names are not validated, and defaults are unknown
to `gd`: a setting at its default is not in the file, so `get` reports it as not set.
