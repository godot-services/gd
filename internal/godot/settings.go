package godot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrUnterminatedValue  = errors.New("unterminated value")
	ErrUnbalancedValue    = errors.New("unbalanced value")
	ErrInvalidSettingPath = errors.New("invalid setting path")
)

type Setting struct{ Path, Raw string }

type blockKind int

const (
	blockOther blockKind = iota // comment or blank line
	blockSection
	blockEntry
)

// block is one unit of the file; text keeps the original bytes including line endings.
type block struct {
	kind blockKind
	text string
	name string // section name, or entry key
	raw  string // entry value as in the file, without the line ending
}

type ProjectSettings struct {
	dir    string
	blocks []block
	eol    string
}

func LoadProjectSettings(dir string) (*ProjectSettings, error) {
	src, err := os.ReadFile(filepath.Join(dir, ProjectFileName))
	if err != nil {
		return nil, err
	}

	s, err := ParseProjectSettings(src)
	if err != nil {
		return nil, err
	}

	s.dir = dir

	return s, nil
}

func ParseProjectSettings(src []byte) (*ProjectSettings, error) {
	text := string(src)
	s := &ProjectSettings{eol: "\n"}
	if i := strings.Index(text, "\n"); i > 0 && text[i-1] == '\r' {
		s.eol = "\r\n"
	}

	for pos, line := 0, 1; pos < len(text); {
		end := lineEnd(text, pos)
		first := text[pos:end]
		b := block{kind: blockOther}

		switch {
		case strings.HasPrefix(first, "["):
			name, _, _ := strings.Cut(first[1:], "]")
			b = block{kind: blockSection, name: strings.TrimSpace(name)}
		case strings.HasPrefix(first, ";") || strings.TrimSpace(first) == "":
		default:
			k := keyEnd(first)
			eq := strings.Index(first[k:], "=")
			if eq < 0 {
				break // not an entry: keep the line verbatim
			}

			eq += k

			var ok bool
			if end, ok = valueEnd(text, pos+eq+1); !ok {
				return nil, fmt.Errorf("%s:%d: %w", ProjectFileName, line, ErrUnterminatedValue)
			}

			b = block{kind: blockEntry, name: strings.TrimSpace(first[:eq])}
			if name, ok := DecodeString(b.name); ok {
				b.name = name // Godot quotes keys with spaces, non-ASCII or =";[]
			}
			b.raw = strings.TrimRight(text[pos+eq+1:end], "\r\n")
		}

		b.text = text[pos:end]
		s.blocks = append(s.blocks, b)
		line += strings.Count(b.text, "\n")
		pos = end
	}

	return s, nil
}

// keyEnd returns the index after a leading quoted key ("..." with \ escapes), else 0.
func keyEnd(line string) int {
	if !strings.HasPrefix(line, `"`) {
		return 0
	}

	for i := 1; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}

	return 0
}

// lineEnd returns the index just after the line ending of the line starting at pos.
func lineEnd(text string, pos int) int {
	if i := strings.Index(text[pos:], "\n"); i >= 0 {
		return pos + i + 1
	}

	return len(text)
}

// valueEnd scans a value starting at start and returns the index just after the line
// ending where its quotes and brackets balance. ok is false if the text ends first.
func valueEnd(text string, start int) (end int, ok bool) {
	depth, inString := 0, false

	for i := start; i < len(text); i++ {
		c := text[i]

		switch {
		case inString && c == '\\':
			i++
		case c == '"':
			inString = !inString
		case inString:
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth--
		case c == '\n' && depth <= 0:
			return i + 1, true
		}
	}

	if !inString && depth <= 0 {
		return len(text), true
	}

	return len(text), false
}

func (s *ProjectSettings) Bytes() []byte {
	var sb strings.Builder
	for _, b := range s.blocks {
		sb.WriteString(b.text)
	}

	return []byte(sb.String())
}

func (s *ProjectSettings) Get(path string) (string, bool) {
	if i := s.find(path); i >= 0 {
		return s.blocks[i].raw, true
	}

	return "", false
}

func (s *ProjectSettings) List() []Setting {
	var out []Setting
	section := ""

	for _, b := range s.blocks {
		switch b.kind {
		case blockSection:
			section = b.name
		case blockEntry:
			path := b.name
			if section != "" {
				path = section + "/" + b.name
			}

			out = append(out, Setting{Path: path, Raw: b.raw})
		}
	}

	return out
}

// find returns the index of the last entry block for path; Godot's loader keeps the last.
func (s *ProjectSettings) find(path string) int {
	section, found := "", -1

	for i, b := range s.blocks {
		switch b.kind {
		case blockSection:
			section = b.name
		case blockEntry:
			p := b.name
			if section != "" {
				p = section + "/" + b.name
			}

			if p == path {
				found = i
			}
		}
	}

	return found
}

// insert puts blocks at index i, first giving a previous line without ending one.
func (s *ProjectSettings) insert(i int, nb ...block) {
	if i > 0 && !strings.HasSuffix(s.blocks[i-1].text, "\n") {
		s.blocks[i-1].text += s.eol
	}

	s.blocks = append(s.blocks[:i], append(nb, s.blocks[i:]...)...)
}

func (s *ProjectSettings) Set(path, raw string) error {
	section, key, found := strings.Cut(path, "/")
	if !found {
		section, key = "", path
	}

	if path == "" || strings.ContainsAny(path, "=[]\"\n\r") || strings.HasPrefix(path, ";") ||
		path != strings.TrimSpace(path) || key == "" || (found && section == "") {
		return fmt.Errorf("%w: %q", ErrInvalidSettingPath, path)
	}

	if end, ok := valueEnd(raw+"\n", 0); !ok || end != len(raw)+1 {
		return fmt.Errorf("%w: %s", ErrUnbalancedValue, path)
	}

	if i := s.find(path); i >= 0 {
		b := &s.blocks[i]
		b.raw = raw
		b.text = b.text[:strings.Index(b.text, "=")+1] + raw + s.eol

		return nil
	}

	keyText := key
	if strings.ContainsFunc(key, func(r rune) bool { return r < 33 || r > 126 || strings.ContainsRune(`=";[]`, r) }) {
		keyText = EncodeString(key)
	}

	e := block{kind: blockEntry, name: key, raw: raw, text: keyText + "=" + raw + s.eol}

	header := -1
	for i, b := range s.blocks {
		if section != "" && b.kind == blockSection && b.name == section {
			header = i
			break
		}
	}

	if section != "" && header < 0 {
		s.insertSection(section, e)
	} else {
		s.insertKey(header, e)
	}

	return nil
}

func (s *ProjectSettings) insertSection(name string, e block) {
	nb := []block{
		{kind: blockSection, name: name, text: "[" + name + "]" + s.eol},
		{text: s.eol}, e, {text: s.eol},
	}

	for i, b := range s.blocks {
		if b.kind == blockSection && b.name > name {
			s.insert(i, nb...)
			return
		}
	}

	nb = nb[:3] // at the end of the file: no trailing blank
	if len(s.blocks) > 0 {
		nb = append([]block{{text: s.eol}}, nb...)
	}

	s.insert(len(s.blocks), nb...)
}

// insertKey adds e to the section whose header is at index header (-1 for the root).
func (s *ProjectSettings) insertKey(header int, e block) {
	next := len(s.blocks)

	for i := header + 1; i < len(s.blocks); i++ {
		if s.blocks[i].kind == blockSection {
			next = i
			break
		}
	}

	at := 0
	for i := header + 1; i < next; i++ {
		if s.blocks[i].kind == blockEntry {
			at = i + 1
		}
	}

	switch {
	case at > 0:
		s.insert(at, e)
	case header < 0:
		s.insert(next, e) // root without entries: before the first section
	default:
		// empty section: after the header and one blank line, keeping the gap to the next
		at = header + 1
		if at < next && strings.TrimSpace(s.blocks[at].text) == "" && s.blocks[at].kind == blockOther {
			at++
			if at == next && next < len(s.blocks) {
				s.insert(at, e, block{text: s.eol})
				return
			}
		}

		s.insert(at, e)
	}
}

func (s *ProjectSettings) Unset(path string) bool {
	i := s.find(path)
	if i < 0 {
		return false
	}

	s.blocks = append(s.blocks[:i], s.blocks[i+1:]...)

	return true
}

// Save writes project.godot atomically, keeping the file mode.
func (s *ProjectSettings) Save() (err error) {
	if s.dir == "" {
		return errors.New("project settings were not loaded from a directory")
	}

	file := filepath.Join(s.dir, ProjectFileName)

	fi, err := os.Stat(file)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(s.dir, ".project.godot-*")
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()

	if _, err = tmp.Write(s.Bytes()); err != nil {
		return err
	}

	if err = tmp.Chmod(fi.Mode().Perm()); err != nil {
		return err
	}

	if err = tmp.Sync(); err != nil {
		return err
	}

	if err = tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), file)
}

// DecodeString decodes a plain "..." value; ok is false for anything else.
func DecodeString(raw string) (string, bool) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", false
	}

	var sb strings.Builder

	body := raw[1 : len(raw)-1]
	for i := 0; i < len(body); i++ {
		switch c := body[i]; {
		case c == '"':
			return "", false
		case c == '\\' && i+1 < len(body):
			i++
			switch body[i] {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '\\', '"':
				sb.WriteByte(body[i])
			default:
				return "", false
			}
		case c == '\\':
			return "", false
		default:
			sb.WriteByte(c)
		}
	}

	return sb.String(), true
}

func EncodeString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// DecodeStringArray decodes PackedStringArray("a", "b"); ok is false for any other value.
func DecodeStringArray(raw string) ([]string, bool) {
	body, ok := strings.CutPrefix(raw, "PackedStringArray(")
	if !ok {
		return nil, false
	}

	body, ok = strings.CutSuffix(strings.TrimSpace(body), ")")
	if !ok {
		return nil, false
	}

	out := []string{}

	for body = strings.TrimSpace(body); body != ""; {
		if body[0] != '"' {
			return nil, false
		}

		end := 1
		for end < len(body) && body[end] != '"' {
			if body[end] == '\\' {
				end++
			}

			end++
		}

		if end >= len(body) {
			return nil, false
		}

		item, ok := DecodeString(body[:end+1])
		if !ok {
			return nil, false
		}

		out = append(out, item)

		body = strings.TrimSpace(body[end+1:])
		if body == "" {
			break
		}

		if body[0] != ',' {
			return nil, false
		}

		body = strings.TrimSpace(body[1:])
	}

	return out, true
}

func EncodeStringArray(items []string) string {
	enc := make([]string, len(items))
	for i, it := range items {
		enc[i] = EncodeString(it)
	}

	return "PackedStringArray(" + strings.Join(enc, ", ") + ")"
}
