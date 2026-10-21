# M5 Chunk A: Project Settings — Design

Status: accepted · Date: 2026-10-21 · Refs: gd#46 · Builds on: [ADRs](../../adr/README.md) (0015)

## Goal

Read and change Godot project settings from the command line, without opening the
editor, and without touching anything in `project.godot` except the setting that
changes:

```sh
gd config get application/config/name
gd config set application/config/name "My Game"
gd config unset application/config/name
gd config list [prefix]
```

The reader/writer is the foundation for later M5 chunks: `gd inspect` (#47), enabling
editor plugins (#51) and `gd doctor` (#50).

## M5 decomposition

| Chunk | Issues   | Content                                                   |
| ----- | -------- | --------------------------------------------------------- |
| A     | #46      | `project.godot` reader/writer, `gd config`                |
| B     | #47, #51 | `gd inspect`, enable/disable editor plugins               |
| C     | #44, #48 | `[scripts]` / `gd run`, `gd build` via export presets      |
| D     | #52, #50 | GDExtension checks, `gd doctor`                           |

One spec → plan → implementation cycle and one PR per chunk, one commit per issue.

## Context

- `project.godot` is written by Godot's `ProjectSettings::save`. It looks like INI, but
  values are Godot variant text: `"strings"` (escapes `\"` and `\\`, newlines kept
  literally), numbers, `true`/`false`, `PackedStringArray("a", "b")`, `Vector2i(1, 2)`,
  arrays, multi-line dictionaries, `Object(InputEventKey,"keycode":0,…)` in `[input]`,
  `&"StringName"`, `^"NodePath"`.
- `gopkg.in/ini.v1` cannot round-trip these values (ADR 0015), so `gd` today only reads
  single lines with regular expressions (`GodotVersion`, `MainScene`).
- A setting path maps to the file like Godot does it: the part before the first `/` is
  the section, the rest is the key (`application/config/name` → `[application]`
  `config/name`). Paths without `/` live before the first section (`config_version`).
- Godot writes sections in alphabetical order. Keys keep their order within a section.
- Defaults of settings are known only to Godot; a setting at its default is not in the
  file.

## Decision: own minimal parser, not the Godot binary

Delegating to `godot --headless --script` (`ProjectSettings.set` + `save`) rewrites the
whole file, needs an installed editor, is slow and may trigger imports. Untouched
content could not stay byte for byte. A full variant parser is not needed either.

`gd` splits the file into blocks and keeps every block's original text:

- **preamble**: comments and blank lines before the first entry or section;
- **section header** `[name]` (plus trailing blank lines/comments up to the next block);
- **entry** `key=value`, where the value ends when it is balanced: outside strings, every
  `(`, `[`, `{` is closed. Strings honour `\` escapes and may span lines. A value still
  open at the end of the file is the error `project.godot:<line>: unterminated value`.

Saving joins the blocks again, so a file loaded and saved without changes is byte for
byte identical.

### Writing

- `set` on an existing entry replaces its value in place.
- A new key is appended to the end of its section (after the last entry, before the
  blank line that separates sections).
- A new section is inserted in alphabetical order among the existing ones, preceded by a
  blank line, like Godot would place it.
- New lines use the file's line ending (CRLF when the first line ends with CRLF, else LF).
- `unset` removes the entry; a section that becomes empty stays.
- The file is written atomically (temp file in the same dir + rename), keeping its mode.
- Nothing is written when loading failed or a value is rejected.

### API (`internal/godot/settings.go`)

```go
type Setting struct{ Path, Raw string }

func LoadProjectSettings(dir string) (*ProjectSettings, error)
func (s *ProjectSettings) Get(path string) (raw string, ok bool)
func (s *ProjectSettings) Set(path, raw string) error // rejects unbalanced raw and invalid paths
func (s *ProjectSettings) Unset(path string) bool
func (s *ProjectSettings) List() []Setting            // file order
func (s *ProjectSettings) Save() error

func DecodeString(raw string) (string, bool)
func EncodeString(s string) string
func DecodeStringArray(raw string) ([]string, bool) // PackedStringArray(...) only
func EncodeStringArray(items []string) string
```

`Project.GodotVersion` and `Project.MainScene` use this reader instead of regular
expressions; their behaviour and errors stay the same.

## `gd config`

Runs in the folder with `project.godot` (same lookup as other commands).

| Command              | Behaviour                                               | Output                     |
| -------------------- | ------------------------------------------------------- | -------------------------- |
| `get <path>`         | strings decoded, everything else raw (multi-line as in the file) | the value          |
| `set <path> <value>` | type inference below                                    | `set <path>`               |
| `unset <path>`       | remove the entry                                        | `unset <path>`             |
| `list [prefix]`      | all entries in file order, filtered by path prefix      | `<path>=<raw>` per entry   |

### Type inference for `set`

1. If the current value is a string, the new value is written as a string, unless the
   new value itself starts with `"` (then it is taken raw).
2. Otherwise these are taken raw: `true`, `false`, `null`, numbers (`-12`, `1.5`, `1e3`),
   and literals starting with `"`, `&"`, `^"`, `[`, `{` or `Name(`.
3. Anything else becomes an escaped, quoted string.
4. A raw value that is not balanced (`[1, 2`) is rejected; the file stays unchanged.

### Errors

- `get`/`unset` of a path not in the file: `<path> is not set in project.godot (Godot uses its default)`, exit 1.
- No `project.godot`: the existing `ErrNoGodotProject`.
- Empty path, or a path with `=`, `[`, `]`, `"`, or a line break: `invalid setting path`.

Output stays plain, in the style of the existing commands, so M7's shared output helper
can migrate it later.

## Testing

- Real `project.godot` files generated with Godot 4.7.2 (fresh project; project with
  input map, autoloads, `[editor_plugins]`, feature overrides like `….mobile`), plus a
  CRLF file and a multi-line string, in `internal/godot/testdata/projects/`.
- Round trip: load + save without changes is byte for byte identical for every sample.
- `set`/`unset`: only the affected entry differs from the original; new section placed
  alphabetically; inference table test; unbalanced values rejected; string escapes.
- `cmd` tests for `get`, `set`, `unset`, `list` like the existing `cmd/*_test.go`.
- Existing `GodotVersion`/`MainScene` tests stay green.

## Documentation

- ADR 0024 "Read and write project.godot with an own minimal parser" (next after M4's 0020–0023;
  renumbered at PR time if main moved on). It also lifts ADR
  0015's "only this line is read" restriction.
- README: "Project settings" section; `docs/configuration-files.md`: short `project.godot`
  section (what `gd` reads and writes).
- Issue #46 adjusted to this design after approval.

## Non-goals

- Editor settings (`editor_settings-4.x.tres`).
- Knowing Godot's default values or validating setting names.
- `--json` output.
- Full variant parsing (only strings and `PackedStringArray` are decoded).
