package godot

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func readSample(t *testing.T, name string) []byte {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", "projects", name))
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func parse(t *testing.T, src string) *ProjectSettings {
	t.Helper()
	s, err := ParseProjectSettings([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func paths(s *ProjectSettings) []string {
	var out []string
	for _, e := range s.List() {
		out = append(out, e.Path)
	}
	return out
}

func TestSettingsRoundTrip(t *testing.T) {
	for _, name := range []string{"fresh.godot", "full.godot", "crlf.godot"} {
		src := readSample(t, name)
		s, err := ParseProjectSettings(src)
		if err != nil {
			t.Fatal(name, err)
		}
		if string(s.Bytes()) != string(src) {
			t.Errorf("%s: round trip differs", name)
		}
	}
}

func TestSettingsGet(t *testing.T) {
	s, err := ParseProjectSettings(readSample(t, "full.godot"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"application/config/name":                    `"Full \"Sample\""`,
		"application/config/description":             "\"line one\nline two with \\\\ backslash\"",
		"config_version":                             "5",
		"editor_plugins/enabled":                     `PackedStringArray("res://addons/foo/plugin.cfg", "res://addons/bar/plugin.cfg")`,
		"rendering/renderer/rendering_method.mobile": `"gl_compatibility"`,
	}
	for k, v := range want {
		if got, ok := s.Get(k); !ok || got != v {
			t.Errorf("Get(%q) = %q, %v; want %q", k, got, ok, v)
		}
	}
	jump, ok := s.Get("input/jump")
	if !ok || !strings.HasPrefix(jump, "{\n\"deadzone\": 0.5,") || !strings.HasSuffix(jump, "]\n}") {
		t.Errorf("input/jump = %q", jump)
	}
	if _, ok := s.Get("application/missing"); ok {
		t.Error("missing key found")
	}
}

func TestSettingsList(t *testing.T) {
	s, err := ParseProjectSettings(readSample(t, "fresh.godot"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"config_version", "application/config/name", "application/config/features", "application/config/icon"}
	if got := paths(s); !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %v", got)
	}
}

func TestSettingsMultiLineStringNotSplit(t *testing.T) {
	s := parse(t, "[a]\nx=\"one\n[b]\ny=2\"\nz=1\n")
	if got, want := paths(s), []string{"a/x", "a/z"}; !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %v", got)
	}
}

func TestSettingsBracketsInStrings(t *testing.T) {
	s := parse(t, "[a]\nx=\"a(;[\"\ny=[\"]\"]\n")
	if got, _ := s.Get("a/x"); got != `"a(;["` {
		t.Errorf("a/x = %q", got)
	}
	if got, _ := s.Get("a/y"); got != `["]"]` {
		t.Errorf("a/y = %q", got)
	}
}

func TestSettingsEscapedBackslash(t *testing.T) {
	s := parse(t, "[a]\nx=\"a\\\\\"\ny=1\n")
	if got, _ := s.Get("a/x"); got != `"a\\"` {
		t.Errorf("a/x = %q", got)
	}
	if got, _ := s.Get("a/y"); got != "1" {
		t.Errorf("a/y = %q", got)
	}
}

func TestSettingsUnterminated(t *testing.T) {
	_, err := ParseProjectSettings([]byte("[a]\nok=1\nx={\n\"k\": 1\n"))
	if !errors.Is(err, ErrUnterminatedValue) {
		t.Fatalf("err = %v", err)
	}
	if err.Error() != "project.godot:3: unterminated value" {
		t.Errorf("message = %q", err.Error())
	}
}

func TestSettingsComments(t *testing.T) {
	src := "; c\n[a]\n; note\nx=1\n"
	s := parse(t, src)
	if got, _ := s.Get("a/x"); got != "1" {
		t.Errorf("a/x = %q", got)
	}
	if got, want := paths(s), []string{"a/x"}; !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %v", got)
	}
	if string(s.Bytes()) != src {
		t.Error("round trip differs")
	}
}

func TestLoadProjectSettings(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ProjectFileName), readSample(t, "crlf.godot"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadProjectSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.eol != "\r\n" {
		t.Errorf("eol = %q", s.eol)
	}
	if got, _ := s.Get("config_version"); got != "5" {
		t.Errorf("config_version = %q", got)
	}
}
