package cmd

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	built := func(v string) *debug.BuildInfo { return &debug.BuildInfo{Main: debug.Module{Version: v}} }
	tests := []struct {
		name    string
		ldflags string
		info    *debug.BuildInfo
		ok      bool
		want    string
	}{
		{"ldflags win", "v0.4.0", built("v0.3.0"), true, "v0.4.0"},
		{"go install", "", built("v0.3.0"), true, "v0.3.0"},
		{"local build", "", built("(devel)"), true, "dev"},
		{"empty build info", "", &debug.BuildInfo{}, true, "dev"},
		{"no build info", "", nil, false, "dev"},
	}
	for _, tt := range tests {
		if got := resolveVersion(tt.ldflags, tt.info, tt.ok); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestVersionFlag(t *testing.T) {
	old := buildVersion
	buildVersion = "v9.9.9"
	t.Cleanup(func() { buildVersion = old })

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "gd version v9.9.9\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestAddVersionFlagIsNotGdVersion(t *testing.T) {
	add, _, err := newRootCmd().Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	if f := add.Flags().Lookup("version"); f == nil || !strings.Contains(f.Usage, "git ref") {
		t.Fatalf("gd add --version is not the git ref flag: %v", f)
	}
}
