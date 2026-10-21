package godot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnterminatedValue = errors.New("unterminated value")

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
			b = block{kind: blockSection, name: strings.TrimSpace(strings.TrimRight(first, "\r\n"))}
			b.name = strings.TrimSuffix(strings.TrimPrefix(b.name, "["), "]")
		case strings.HasPrefix(first, ";") || strings.TrimSpace(first) == "":
		default:
			eq := strings.Index(first, "=")
			if eq < 0 {
				break // not an entry: keep the line verbatim
			}

			var ok bool
			if end, ok = valueEnd(text, pos+eq+1); !ok {
				return nil, fmt.Errorf("%s:%d: %w", ProjectFileName, line, ErrUnterminatedValue)
			}

			b = block{kind: blockEntry, name: strings.TrimSpace(first[:eq])}
			b.raw = strings.TrimRight(text[pos+eq+1:end], "\r\n")
		}

		b.text = text[pos:end]
		s.blocks = append(s.blocks, b)
		line += strings.Count(b.text, "\n")
		pos = end
	}

	return s, nil
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
	for _, e := range s.List() {
		if e.Path == path {
			return e.Raw, true
		}
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
