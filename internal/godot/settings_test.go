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
	for _, name := range []string{"fresh.godot", "full.godot", "crlf.godot", "quoted.godot"} {
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

func setOn(t *testing.T, src []byte, path, raw string) string {
	t.Helper()
	s, err := ParseProjectSettings(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Set(path, raw); err != nil {
		t.Fatal(err)
	}
	return string(s.Bytes())
}

func TestSetReplacesInPlace(t *testing.T) {
	src := readSample(t, "full.godot")
	got := setOn(t, src, "display/window/size/viewport_width", "1920")
	want := strings.Replace(string(src), "viewport_width=1280", "viewport_width=1920", 1)
	if got != want {
		t.Errorf("got %q", got)
	}
}

func TestSetMultiLineInPlace(t *testing.T) {
	src := string(readSample(t, "full.godot"))
	got := setOn(t, []byte(src), "input/jump", "{}")
	i := strings.Index(src, "jump={")
	j := strings.Index(src, "\n}\n") + 3
	want := src[:i] + "jump={}\n" + src[j:]
	if got != want {
		t.Errorf("got %q", got)
	}
}

func TestSetNewKeyAppendsToSection(t *testing.T) {
	src := string(readSample(t, "fresh.godot"))
	got := setOn(t, []byte(src), "application/run/main_scene", `"res://m.tscn"`)
	want := src + "run/main_scene=\"res://m.tscn\"\n"
	if got != want {
		t.Errorf("got %q", got)
	}
}

func TestSetNewSectionAlphabetical(t *testing.T) {
	src := string(readSample(t, "full.godot"))
	got := setOn(t, []byte(src), "gui/theme/custom", `"res://t.tres"`)
	want := strings.Replace(src, "[input]", "[gui]\n\ntheme/custom=\"res://t.tres\"\n\n[input]", 1)
	if got != want {
		t.Errorf("got %q", got)
	}
	got = setOn(t, []byte(src), "zzz/a", "1")
	if got != src+"\n[zzz]\n\na=1\n" {
		t.Errorf("got %q", got)
	}
}

func TestSetRootKey(t *testing.T) {
	src := string(readSample(t, "fresh.godot"))
	got := setOn(t, []byte(src), "foo", "1")
	want := strings.Replace(src, "config_version=5\n", "config_version=5\nfoo=1\n", 1)
	if got != want {
		t.Errorf("got %q", got)
	}
	if got := setOn(t, []byte("; c\n\n[a]\nx=1\n"), "foo", "1"); got != "; c\n\nfoo=1\n[a]\nx=1\n" {
		t.Errorf("got %q", got)
	}
	if got := setOn(t, nil, "foo", "1"); got != "foo=1\n" {
		t.Errorf("got %q", got)
	}
}

func TestSetCRLF(t *testing.T) {
	got := setOn(t, readSample(t, "crlf.godot"), "application/run/main_scene", `"x"`)
	if !strings.Contains(got, "run/main_scene=\"x\"\r\n") {
		t.Errorf("got %q", got)
	}
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("bare LF in %q", got)
	}
}

func TestSetNoTrailingNewline(t *testing.T) {
	if got := setOn(t, []byte("[a]\nx=1"), "a/y", "2"); got != "[a]\nx=1\ny=2\n" {
		t.Errorf("got %q", got)
	}
}

func TestSetEmptyFile(t *testing.T) {
	if got := setOn(t, nil, "application/config/name", `"G"`); got != "[application]\n\nconfig/name=\"G\"\n" {
		t.Errorf("got %q", got)
	}
}

func TestSetRejectsUnbalanced(t *testing.T) {
	s := parse(t, "[a]\nx=1\n")
	for _, raw := range []string{"[1, 2", "\"open"} {
		if err := s.Set("a/x", raw); !errors.Is(err, ErrUnbalancedValue) {
			t.Errorf("%q: err = %v", raw, err)
		}
	}
	if string(s.Bytes()) != "[a]\nx=1\n" {
		t.Error("bytes changed")
	}
}

func TestSetInvalidPath(t *testing.T) {
	s := parse(t, "")
	for _, p := range []string{"", "a=b", "[a]/b", "a/\"b", "a/b\nc", "a/b\rc", "a/", "/x", ";a", " a/b", "a/b "} {
		if err := s.Set(p, "1"); !errors.Is(err, ErrInvalidSettingPath) {
			t.Errorf("%q: err = %v", p, err)
		}
	}
}

func TestSetDuplicateKeyFirstWins(t *testing.T) {
	if got := setOn(t, []byte("[a]\nx=1\nx=2\n"), "a/x", "3"); got != "[a]\nx=3\nx=2\n" {
		t.Errorf("got %q", got)
	}
}

func TestSectionHeaderWithComment(t *testing.T) {
	s := parse(t, "[a] ; c\nx=1\n")
	if v, ok := s.Get("a/x"); !ok || v != "1" {
		t.Errorf("got %q %v", v, ok)
	}
}

func TestUnset(t *testing.T) {
	s := parse(t, string(readSample(t, "full.godot")))
	if !s.Unset("autoload/Global") {
		t.Fatal("not removed")
	}
	if !strings.Contains(string(s.Bytes()), "[autoload]\n\n\n[display]") || strings.Contains(string(s.Bytes()), "Global") {
		t.Errorf("got %q", s.Bytes())
	}
	if s.Unset("autoload/Global") {
		t.Error("second unset true")
	}
}

func TestSaveAtomic(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ProjectFileName)
	if err := os.WriteFile(file, readSample(t, "full.godot"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadProjectSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Set("display/window/size/viewport_width", "1"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(file)
	if string(got) != string(s.Bytes()) {
		t.Error("content differs")
	}
	if fi, _ := os.Stat(file); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v", fi.Mode())
	}
	if ents, _ := os.ReadDir(dir); len(ents) != 1 {
		t.Errorf("%d files in dir", len(ents))
	}
}

func TestStringCodec(t *testing.T) {
	enc := EncodeString("a\"b\\c\nd")
	if enc != "\"a\\\"b\\\\c\nd\"" {
		t.Errorf("enc = %q", enc)
	}
	if got, ok := DecodeString(enc); !ok || got != "a\"b\\c\nd" {
		t.Errorf("dec = %q %v", got, ok)
	}
	if _, ok := DecodeString("12"); ok {
		t.Error("12 decoded")
	}
}

func TestStringArrayCodec(t *testing.T) {
	got, ok := DecodeStringArray(`PackedStringArray("res://a", "b\"c")`)
	if !ok || !reflect.DeepEqual(got, []string{"res://a", `b"c`}) {
		t.Errorf("got %q %v", got, ok)
	}
	if got, ok := DecodeStringArray("PackedStringArray()"); !ok || len(got) != 0 {
		t.Errorf("empty: %q %v", got, ok)
	}
	if EncodeStringArray(nil) != "PackedStringArray()" {
		t.Error("encode empty")
	}
	if EncodeStringArray([]string{"a", "b"}) != `PackedStringArray("a", "b")` {
		t.Error("encode")
	}
	if _, ok := DecodeStringArray("[1]"); ok {
		t.Error("[1] decoded")
	}
}

func TestDecodeStringUnknownEscape(t *testing.T) {
	if _, ok := DecodeString(`"a\rb"`); ok {
		t.Error("\\r decoded")
	}
	if _, ok := DecodeStringArray(`PackedStringArray("a\rb")`); ok {
		t.Error("array with \\r decoded")
	}
}

func TestSetEmptySectionKeepsGap(t *testing.T) {
	if got := setOn(t, []byte("[a]\n\n[b]\nx=1\n"), "a/k", "1"); got != "[a]\n\nk=1\n\n[b]\nx=1\n" {
		t.Errorf("got %q", got)
	}
}

func TestQuotedKeys(t *testing.T) {
	s := parse(t, string(readSample(t, "quoted.godot")))
	want := []string{"config_version", "application/config/name", "application/config/features", "autoload/Größe", "input/Move Left", "input/jump"}
	if got := paths(s); !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %q", got)
	}
	if v, ok := s.Get("autoload/Größe"); !ok || v != `"*res://groesse.gd"` {
		t.Errorf("autoload = %q %v", v, ok)
	}
	if v, ok := s.Get("input/Move Left"); !ok || !strings.HasPrefix(v, "{\n\"deadzone\": 0.2,") {
		t.Errorf("Move Left = %q %v", v, ok)
	}
	if v, ok := parse(t, "[x]\n\"a=b\"=1\n").Get("x/a=b"); !ok || v != "1" {
		t.Errorf("a=b = %q %v", v, ok)
	}
}

func TestSetQuotedKeys(t *testing.T) {
	src := string(readSample(t, "quoted.godot"))
	got := setOn(t, []byte(src), "input/Move Left", "{}")
	want := strings.Replace(src, "\"Move Left\"={\n\"deadzone\": 0.2,\n\"events\": []\n}\n", "\"Move Left\"={}\n", 1)
	if got != want {
		t.Errorf("existing: got %q", got)
	}
	got = setOn(t, []byte(src), "input/Jump Up", "{}")
	if got != src+"\"Jump Up\"={}\n" {
		t.Errorf("new: got %q", got)
	}
	got = setOn(t, []byte(src), "autoload/Ünï", `"*res://u.gd"`)
	if !strings.Contains(got, "\"Größe\"=\"*res://groesse.gd\"\n\"Ünï\"=\"*res://u.gd\"\n") {
		t.Errorf("non-ASCII: got %q", got)
	}
	if got := setOn(t, nil, "a/b;c", "1"); got != "[a]\n\n\"b;c\"=1\n" {
		t.Errorf("semicolon: got %q", got)
	}
}
