package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/godot-services/gd/internal/godot"
)

func TestInferValue(t *testing.T) {
	tests := []struct {
		current string
		has     bool
		arg     string
		want    string
	}{
		{"", false, "My Game", `"My Game"`},
		{"", false, "1280", "1280"},
		{"", false, "-1.5e3", "-1.5e3"},
		{"", false, "true", "true"},
		{"", false, "null", "null"},
		{"", false, `PackedStringArray("4.3")`, `PackedStringArray("4.3")`},
		{"", false, "[1, 2]", "[1, 2]"},
		{"", false, "{}", "{}"},
		{"", false, `&"name"`, `&"name"`},
		{"", false, `^"path"`, `^"path"`},
		{"", false, `"q"`, `"q"`},
		{`"Old"`, true, "123", `"123"`},
		{`"Old"`, true, `"x"`, `"x"`},
		{"5", true, "abc", `"abc"`},
		{"", false, `say "hi"`, `"say \"hi\""`},
		{"", false, "1.2.3", `"1.2.3"`},
		{"", false, "[WIP] my game", `"[WIP] my game"`},
		{"", false, `"q" extra`, `"\"q\" extra"`},
		{"", false, "[1] [2]", `"[1] [2]"`},
		{"", false, "Vector2i(1, 2) x", `"Vector2i(1, 2) x"`},
		{"", false, `&"a"b`, `"&\"a\"b"`},
		{"", false, `{"a": 1}`, `{"a": 1}`},
		{"", false, `["]", 1]`, `["]", 1]`},
		{"", false, "[1, 2", "[1, 2"},
		{`"Old"`, true, `"q" extra`, `"\"q\" extra"`},
	}
	for _, tt := range tests {
		if got := inferValue(tt.current, tt.has, tt.arg); got != tt.want {
			t.Errorf("inferValue(%q, %v, %q) = %q, want %q", tt.current, tt.has, tt.arg, got, tt.want)
		}
	}
}

func writeProject(t *testing.T, content string) string {
	t.Helper()
	dir := newTestProject(t)
	if err := os.WriteFile(filepath.Join(dir, "project.godot"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func readProject(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "project.godot"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestConfigGetString(t *testing.T) {
	writeProject(t, "[application]\nconfig/name=\"My \\\"G\\\"\"\n")
	out, err := runOut(t, newFakeLibrary(t), "config", "get", "application/config/name")
	if err != nil || out != "My \"G\"\n" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestConfigGetRaw(t *testing.T) {
	writeProject(t, "[application]\nconfig/features=PackedStringArray(\"4.7\", \"Forward Plus\")\n")
	out, err := runOut(t, newFakeLibrary(t), "config", "get", "application/config/features")
	if err != nil || out != "PackedStringArray(\"4.7\", \"Forward Plus\")\n" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestConfigGetMissing(t *testing.T) {
	newTestProject(t)
	_, err := runOut(t, newFakeLibrary(t), "config", "get", "application/x")
	if !errors.Is(err, errSettingNotSet) || err.Error() != "application/x is not set in project.godot (Godot uses its default)" {
		t.Fatalf("err=%v", err)
	}
}

func TestConfigSet(t *testing.T) {
	dir := newTestProject(t)
	out, err := runOut(t, newFakeLibrary(t), "config", "set", "application/config/name", "My Game")
	if err != nil || out != "set application/config/name\n" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if got := readProject(t, dir); got != "[application]\n\nconfig/name=\"My Game\"\n" {
		t.Fatalf("file=%q", got)
	}
}

func TestConfigSetInvalidKeepsFile(t *testing.T) {
	const orig = "[a]\n\nx=1\n"
	dir := writeProject(t, orig)
	lib := newFakeLibrary(t)
	if _, err := runOut(t, lib, "config", "set", "a/x", "[1, 2"); err == nil {
		t.Error("unbalanced value accepted")
	}
	if _, err := runOut(t, lib, "config", "set", "a=b/x", "1"); err == nil {
		t.Error("invalid path accepted")
	}
	if got := readProject(t, dir); got != orig {
		t.Fatalf("file changed: %q", got)
	}
}

func TestConfigUnset(t *testing.T) {
	dir := writeProject(t, "[a]\n\nx=1\n")
	lib := newFakeLibrary(t)
	out, err := runOut(t, lib, "config", "unset", "a/x")
	if err != nil || out != "unset a/x\n" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if got := readProject(t, dir); got != "[a]\n\n" {
		t.Fatalf("file=%q", got)
	}
	if _, err := runOut(t, lib, "config", "unset", "a/x"); !errors.Is(err, errSettingNotSet) {
		t.Fatalf("err=%v", err)
	}
}

func TestConfigList(t *testing.T) {
	src, err := os.ReadFile("../internal/godot/testdata/projects/fresh.godot")
	if err != nil {
		t.Fatal(err)
	}
	writeProject(t, string(src))
	lib := newFakeLibrary(t)
	out, err := runOut(t, lib, "config", "list")
	want := "config_version=5\napplication/config/name=\"Fresh\"\napplication/config/features=PackedStringArray(\"4.7\", \"Forward Plus\")\napplication/config/icon=\"res://icon.svg\"\n"
	if err != nil || out != want {
		t.Fatalf("out=%q err=%v", out, err)
	}
	out, err = runOut(t, lib, "config", "list", "application/config/f")
	if err != nil || out != "application/config/features=PackedStringArray(\"4.7\", \"Forward Plus\")\n" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestConfigNoProject(t *testing.T) {
	chdir(t, t.TempDir())
	_, err := runOut(t, newFakeLibrary(t), "config", "list")
	if !errors.Is(err, godot.ErrNoGodotProject) {
		t.Fatalf("err=%v", err)
	}
}

func TestConfigDataGoesToStdout(t *testing.T) {
	writeProject(t, "[a]\n\nx=\"v\"\n")
	for _, args := range [][]string{{"config", "get", "a/x"}, {"config", "list"}} {
		var out, errBuf bytes.Buffer
		root := newRootCmd()
		root.SetArgs(append([]string{"--library-url", "http://unused"}, args...))
		root.SetOut(&out)
		root.SetErr(&errBuf)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.Len() == 0 || errBuf.Len() != 0 {
			t.Errorf("%v: out=%q err=%q", args, out.String(), errBuf.String())
		}
	}
}
