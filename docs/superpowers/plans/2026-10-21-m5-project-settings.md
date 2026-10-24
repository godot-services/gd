# M5 Chunk A: Project Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A byte-preserving reader/writer for `project.godot` in `internal/godot` and the command `gd config get|set|unset|list`.

**Architecture:** `project.godot` is split into blocks (preamble/other lines, section headers, entries) that keep their original text; an entry's value ends when strings and brackets are balanced. Saving joins the blocks, so only changed entries differ. `cmd/config.go` adds type inference on top.

**Tech Stack:** Go, cobra, standard library only (no new dependencies).

**Spec:** `docs/superpowers/specs/2026-10-21-m5-project-settings-design.md`

## Global Constraints

- No new dependencies; `gopkg.in/ini.v1` is not used for `project.godot`.
- A file loaded and saved without changes is byte for byte identical.
- Path mapping: part before the first `/` is the section, the rest the key; no `/` → root section `""` (before the first `[section]`).
- New key: appended after the last entry of its section. New section: inserted in alphabetical order among existing sections, written as `[name]` + blank line + entry, separated from the previous block by a blank line (Godot's layout).
- New lines use CRLF when the file's first line ends with `\r\n`, else LF.
- `unset` removes only the entry; empty sections stay.
- Save is atomic: temp file in the same dir, then rename; file mode kept.
- Errors (exact text): `project.godot:<line>: unterminated value`; `invalid setting path` (empty, or containing `=`, `[`, `]`, `"`, `\n`, `\r`); `<path> is not set in project.godot (Godot uses its default)`.
- Output of commands: `get` prints the value (strings decoded); `set <path>`; `unset <path>`; `list` prints `<path>=<raw>` per entry.
- Commits: one per task, message ends with `Refs #46` (last task `Closes #46`) and `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- English in code, docs and commits.

## Review Focus

1. A multi-line string value containing lines that look like `[input]` or `key=value` must stay inside its entry (Task 1 test `TestSettingsMultiLineStringNotSplit`).
2. Brackets and `;` inside strings (`"a(;["`) must not affect balance or be treated as comments (Task 1 test `TestSettingsBracketsInStrings`).
3. An escaped backslash before a closing quote (`"a\\"`) ends the string (Task 1 test `TestSettingsEscapedBackslash`).
4. A file without trailing newline (and an empty file, which `gd`'s cmd tests create) must still get well-formed new entries (Task 2 tests `TestSetNoTrailingNewline`, `TestSetEmptyFile`).
5. `set` with a rejected value or invalid path must leave the file untouched (Task 2 test `TestSetRejectsUnbalanced`, Task 4 test `TestConfigSetInvalidKeepsFile`).

---

### Task 1: Parse project.godot into blocks (read side)

**Files:**
- Create: `internal/godot/settings.go`
- Create: `internal/godot/settings_test.go`
- Create: `.gitattributes` with `internal/godot/testdata/** -text` (keeps the CRLF sample intact)
- Testdata (already generated with Godot 4.7.2, add to git): `internal/godot/testdata/projects/fresh.godot`, `full.godot`, `crlf.godot`

**Interfaces:**
- Produces:
  ```go
  const ProjectFileName // existing, "project.godot"
  var ErrUnterminatedValue = errors.New("unterminated value") // wrapped as "project.godot:<line>: unterminated value"
  type Setting struct{ Path, Raw string }
  type ProjectSettings struct { /* dir string; blocks []block; eol string */ }
  func LoadProjectSettings(dir string) (*ProjectSettings, error)
  func ParseProjectSettings(src []byte) (*ProjectSettings, error) // used by Load and tests
  func (s *ProjectSettings) Bytes() []byte                       // joined block text
  func (s *ProjectSettings) Get(path string) (raw string, ok bool)
  func (s *ProjectSettings) List() []Setting                      // file order
  ```
  Raw is the value text exactly as in the file (after `=`, without the line ending).

- [ ] **Step 1: Write the failing tests**

```go
func TestSettingsRoundTrip(t *testing.T) // for fresh/full/crlf: ParseProjectSettings(src).Bytes() == src
func TestSettingsGet(t *testing.T) {
	// full.godot
	// Get("application/config/name") == `"Full \"Sample\""`
	// Get("application/config/description") == "\"line one\nline two with \\\\ backslash\""
	// Get("config_version") == "5"
	// Get("editor_plugins/enabled") == `PackedStringArray("res://addons/foo/plugin.cfg", "res://addons/bar/plugin.cfg")`
	// Get("input/jump") starts with "{\n\"deadzone\": 0.5," and ends with "]\n}"
	// Get("rendering/renderer/rendering_method.mobile") == `"gl_compatibility"`
	// Get("application/missing") ok == false
}
func TestSettingsList(t *testing.T) // fresh.godot: paths == [config_version application/config/name application/config/features application/config/icon]
func TestSettingsMultiLineStringNotSplit(t *testing.T) // src "[a]\nx=\"one\n[b]\ny=2\"\nz=1\n": List paths == [a/x a/z]
func TestSettingsBracketsInStrings(t *testing.T)       // "[a]\nx=\"a(;[\"\ny=[\"]\"]\n": Get a/x == `"a(;["`, a/y == `["]"]`
func TestSettingsEscapedBackslash(t *testing.T)        // "[a]\nx=\"a\\\\\"\ny=1\n": Get a/x == `"a\\"`, a/y == "1"
func TestSettingsUnterminated(t *testing.T)            // "[a]\nok=1\nx={\n\"k\": 1\n": errors.Is(err, ErrUnterminatedValue), err.Error() == "project.godot:3: unterminated value"
func TestSettingsComments(t *testing.T)                // "; c\n[a]\n; note\nx=1\n": Get a/x == "1", List paths == [a/x], round trip holds
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/godot/ -run 'TestSettings' -v`
Expected: FAIL (undefined: ParseProjectSettings)

- [ ] **Step 3: Implement the parser in `internal/godot/settings.go`**

Line scan: a line starting with `[` (outside a value) is a section header; a line starting with `;` or blank is an "other" block; any other line starts an entry: key = text before the first `=`, value continues across lines until balanced. Balance: track `"` strings with `\` escapes (an escape skips the next byte); outside strings count `(`/`[`/`{` up and `)`/`]`/`}` down; balanced when depth is 0 and not in a string at a line end. Each block keeps its full text including the line ending. `eol` = `"\r\n"` if the first line ends with it, else `"\n"`. Raw strips the trailing `\n`/`\r\n` only.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/godot/ -v`
Expected: PASS (existing tests too)

- [ ] **Step 5: Commit**

```bash
git add .gitattributes internal/godot/settings.go internal/godot/settings_test.go internal/godot/testdata
git commit -m "Parse project.godot into blocks

Refs #46

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 2: Write settings (Set, Unset, Save, string helpers)

**Files:**
- Modify: `internal/godot/settings.go`
- Modify: `internal/godot/settings_test.go`

**Interfaces:**
- Consumes: Task 1 types.
- Produces:
  ```go
  var ErrInvalidSettingPath = errors.New("invalid setting path")
  var ErrUnbalancedValue = errors.New("unbalanced value")
  func (s *ProjectSettings) Set(path, raw string) error
  func (s *ProjectSettings) Unset(path string) bool
  func (s *ProjectSettings) Save() error // writes <dir>/project.godot atomically
  func DecodeString(raw string) (string, bool)          // `"a\"b\\c"` -> `a"b\c`; false if raw is not a plain "..." string
  func EncodeString(s string) string                    // escapes \ and ", keeps newlines literally
  func DecodeStringArray(raw string) ([]string, bool)   // PackedStringArray("a", "b") -> [a b]; PackedStringArray() -> []
  func EncodeStringArray(items []string) string         // -> PackedStringArray("a", "b")
  ```

- [ ] **Step 1: Write the failing tests**

```go
func TestSetReplacesInPlace(t *testing.T)   // full.godot: Set("display/window/size/viewport_width","1920"); Bytes() == src with "viewport_width=1280" replaced by "viewport_width=1920"
func TestSetMultiLineInPlace(t *testing.T)  // full.godot: Set("input/jump", "{}"); result == src with the whole jump={...} entry replaced by "jump={}\n"
func TestSetNewKeyAppendsToSection(t *testing.T) // fresh.godot: Set("application/run/main_scene", `"res://m.tscn"`) -> new line after config/icon line
func TestSetNewSectionAlphabetical(t *testing.T) // full.godot: Set("gui/theme/custom", `"res://t.tres"`) -> "[gui]\n\ntheme/custom=\"res://t.tres\"\n\n" inserted right before "[input]"; Set("zzz/a","1") -> appended at end as "\n[zzz]\n\na=1\n"
func TestSetRootKey(t *testing.T)           // fresh.godot: Set("foo","1") -> "foo=1\n" right after "config_version=5\n"
func TestSetCRLF(t *testing.T)              // crlf.godot: Set("application/run/main_scene", `"x"`) -> new line ends with "\r\n"; no bare "\n" in output
func TestSetNoTrailingNewline(t *testing.T) // "[a]\nx=1": Set("a/y","2") -> "[a]\nx=1\ny=2\n"
func TestSetEmptyFile(t *testing.T)         // "": Set("application/config/name", `"G"`) -> "[application]\n\nconfig/name=\"G\"\n"
func TestSetRejectsUnbalanced(t *testing.T) // Set("a/x","[1, 2") -> ErrUnbalancedValue; Set("a/x","\"open") -> ErrUnbalancedValue; Bytes unchanged
func TestSetInvalidPath(t *testing.T)       // "", "a=b", "[a]/b", "a/\"b", "a/b\nc" -> ErrInvalidSettingPath
func TestUnset(t *testing.T)                // full.godot: Unset("autoload/Global") == true; "[autoload]\n\n" header stays, entry gone; Unset("autoload/Global") == false
func TestSaveAtomic(t *testing.T)           // LoadProjectSettings(dir) on a copy of full.godot with mode 0o600, Set, Save -> file content == Bytes(), mode still 0o600, no extra files in dir
func TestStringCodec(t *testing.T)          // EncodeString("a\"b\\c\nd") == "\"a\\\"b\\\\c\nd\""; DecodeString of it round-trips; DecodeString("12") false
func TestStringArrayCodec(t *testing.T)     // Decode(`PackedStringArray("res://a", "b\"c")`) == [res://a b"c]; Encode([]) == "PackedStringArray()"; Decode("[1]") false
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/godot/ -run 'TestSet|TestUnset|TestSave|Codec' -v`
Expected: FAIL (undefined: Set)

- [ ] **Step 3: Implement**

Set validates the path, then checks the raw value with the same balance scanner as the parser (must be balanced and not end inside a string). Existing entry: replace the block text with `key=raw+eol` keeping the key text. New key: insert after the section's last entry block, or after the header and the "other" blocks directly following it when the section has none; if the preceding block lacks a line ending, add `eol` to it first. New section: insert before the first header whose name sorts after it, as blocks `[name]eol`, `eol`, entry, `eol` (the trailing blank keeps the gap to the next section); at the end of the file instead prefix with `eol` when the file is non-empty and drop the trailing blank. Save: `os.CreateTemp(dir, ".project.godot-*")`, write, `Chmod` to the old mode, `Rename`; remove the temp file on every error.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/godot/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/godot/settings.go internal/godot/settings_test.go
git commit -m "Set and unset project settings byte for byte

Refs #46

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task 3: GodotVersion and MainScene use the reader

**Files:**
- Modify: `internal/godot/project.go` (drop `mainSceneRe`, `featuresRe`, `quotedRe`; keep `godotVersionRe`)
- Test: `internal/godot/project_test.go`

**Interfaces:**
- Consumes: `LoadProjectSettings`, `Get`, `DecodeString`, `DecodeStringArray`.
- Produces: unchanged `(*Project).GodotVersion() (string, error)`, `(*Project).MainScene() (string, error)`.

- [ ] **Step 1: Write the failing test**

Add a case to `TestGodotVersion`: `{"features after multi-line string", "[application]\nconfig/description=\"x\nconfig/features=PackedStringArray(\\\"9.9\\\")\"\nconfig/features=PackedStringArray(\"4.5\")\n", "4.5", nil}` (today's regex returns the wrong value from inside the string, or none), and `TestMainSceneFromFull` reading `testdata/projects/full.godot` → `res://main.tscn`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/godot/ -run 'TestGodotVersion|TestMainScene' -v`
Expected: FAIL on the new case

- [ ] **Step 3: Implement** — `GodotVersion`: `DecodeStringArray(Get("application/config/features"))`, first item matching `godotVersionRe`, else `ErrNoGodotVersion`. `MainScene`: `DecodeString(Get("application/run/main_scene"))`, empty or missing → `ErrNoMainScene`. Parse errors are returned. Remove the stale "only this line is read" comment.

- [ ] **Step 4: Run all tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Commit** (`Read project.godot through the settings reader`, `Refs #46`, trailer)

### Task 4: `gd config get|set|unset|list`

**Files:**
- Create: `cmd/config.go`, `cmd/config_test.go`
- Modify: `cmd/root.go` (register `newConfigCmd()`)

**Interfaces:**
- Consumes: `godot.NewProject(".")`, `godot.LoadProjectSettings`, `Get/Set/Unset/List/Save`, `DecodeString`, `EncodeString`.
- Produces:
  ```go
  var errSettingNotSet = errors.New("is not set in project.godot (Godot uses its default)") // error text: "<path> " + this
  func newConfigCmd() *cobra.Command
  func inferValue(current string, hasCurrent bool, arg string) string
  ```

- [ ] **Step 1: Write the failing tests** (use `newTestProject`, `runOut(t, newFakeLibrary(t), "config", …)`, write `project.godot` content as needed)

```go
func TestInferValue(t *testing.T) // table (current, has, arg) -> want:
	// ("", false, "My Game") -> `"My Game"`; ("", false, "1280") -> "1280"; ("", false, "-1.5e3") -> "-1.5e3"
	// ("", false, "true") -> "true"; ("", false, "null") -> "null"
	// ("", false, `PackedStringArray("4.3")`) -> same; ("", false, "[1, 2]") -> same; ("", false, "{}") -> same
	// ("", false, `&"name"`) -> same; ("", false, `^"path"`) -> same; ("", false, `"q"`) -> `"q"`
	// (`"Old"`, true, "123") -> `"123"`; (`"Old"`, true, `"x"`) -> `"x"`; ("5", true, "abc") -> `"abc"`
	// ("", false, `say "hi"`) -> `"say \"hi\""`; ("", false, "1.2.3") -> `"1.2.3"`
func TestConfigGetString(t *testing.T)   // project.godot "[application]\nconfig/name=\"My \\\"G\\\"\"\n" -> output `My "G"` + "\n"
func TestConfigGetRaw(t *testing.T)      // features -> `PackedStringArray("4.7", "Forward Plus")` + "\n"
func TestConfigGetMissing(t *testing.T)  // errors.Is(err, errSettingNotSet); err.Error() == "application/x is not set in project.godot (Godot uses its default)"
func TestConfigSet(t *testing.T)         // empty project.godot; set application/config/name "My Game" -> output "set application/config/name\n"; file == "[application]\n\nconfig/name=\"My Game\"\n"
func TestConfigSetInvalidKeepsFile(t *testing.T) // set a/x "[1, 2" -> error; set "a=b/x" 1 -> error; file unchanged
func TestConfigUnset(t *testing.T)       // output "unset a/x\n"; entry gone; unset again -> errSettingNotSet
func TestConfigList(t *testing.T)        // fresh sample; "list" -> 4 lines "config_version=5\napplication/config/name=\"Fresh\"\n…"; "list application/config/f" -> only the features line
func TestConfigNoProject(t *testing.T)   // temp dir without project.godot -> errors.Is(err, godot.ErrNoGodotProject)
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -run 'TestConfig|TestInferValue' -v`
Expected: FAIL (undefined: newConfigCmd)

- [ ] **Step 3: Implement** — `config` parent with four subcommands (`get <path>` ExactArgs(1), `set <path> <value>` ExactArgs(2), `unset <path>` ExactArgs(1), `list [prefix]` MaximumNArgs(1)). `get`: print `DecodeString` result when ok, else raw. Numbers: `^-?\d+(\.\d+)?([eE][+-]?\d+)?$`; constructor literal: `^[A-Za-z_][A-Za-z0-9_]*\(`. Invalid raw values are rejected by `Set`, so `inferValue` does not validate. Output via `cmd.Printf`/`cmd.Println` like other commands.

- [ ] **Step 4: Run all tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Manual check against real Godot**

Run in a temp copy of `full.godot` as `project.godot`: `go run . config set display/window/size/viewport_width 1920` then `"<editor path from gd editor list>" --headless --path <dir> --quit` exits 0 and `git diff --no-index` against the original shows only that line.

- [ ] **Step 6: Commit** (`Add gd config get, set, unset and list`, `Refs #46`, trailer)

### Task 5: ADR and documentation

**Files:**
- Create: `docs/adr/0024-project-godot-reader-writer.md` (from `docs/adr/template.md`; next free number at commit time, see `side-session-notes/.../m5-adr-numbers.md`; refs gd#46, spec, ADR 0015)
- Modify: `docs/adr/README.md` (row 0024; mark 0015 as "amended by 0024")
- Modify: `README.md` (section "Project settings" after "Export templates" with the four commands; row `config` in "shortcut commands")
- Modify: `docs/configuration-files.md` (section `project.godot`: what `gd` reads — config/features, run/main_scene — and that `gd config` writes only the changed entry)

ADR content: Context (variant syntax, ini.v1 cannot round-trip, editor rewrite loses layout); Decision (own block parser, balance rule, path mapping, placement of new keys/sections, line endings, atomic write, type inference rules 1–4, decode only strings and PackedStringArray, replaces single-line regex reads from 0015); Consequences (byte-preserving, no editor needed; values outside the inference rules need Godot syntax; unknown setting names are not validated; defaults unknown).

- [ ] **Step 1: Write ADR, README, configuration docs**
- [ ] **Step 2: Verify** — `go test ./... && go vet ./...` PASS; README commands match `go run . config --help`
- [ ] **Step 3: Commit** (`Document gd config and the project.godot parser`, body `Closes #46`, trailer)
