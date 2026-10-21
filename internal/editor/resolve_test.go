package editor

import (
	"errors"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	ed := func(name, v, build, src string, dn bool) Editor {
		return Editor{Name: name, Version: v, Build: build, Source: src, Dotnet: dn}
	}
	reg := &Registry{Default: "def", Editors: map[string]Editor{
		"def":    ed("def", "4.6-stable", "official", "linked", false),
		"4.7.2":  ed("4.7.2", "4.7.2-stable", "official", "managed", false),
		"b":      ed("b", "4.8-stable", "official", "linked", false),
		"a":      ed("a", "4.8-stable", "official", "linked", false),
		"4.8":    ed("4.8", "4.8-stable", "official", "managed", false),
		"4.7.2d": ed("4.7.2d", "4.7.2-stable", "official", "managed", true),
		"cust":   ed("cust", "4.9-stable", "custom", "linked", false),
	}}
	tests := []struct {
		name string
		req  Requirement
		want string
	}{
		{"local wins incl custom", Requirement{Version: "4.7.2-stable", Local: "cust"}, "cust"},
		{"default when matching", Requirement{Version: "4.6-stable"}, "def"},
		{"managed", Requirement{Version: "4.7.2-stable"}, "4.7.2"},
		{"dotnet flavour", Requirement{Version: "4.7.2-stable", Dotnet: true}, "4.7.2d"},
		{"managed before linked", Requirement{Version: "4.8-stable"}, "4.8"},
		{"no requirement uses default", Requirement{}, "def"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if e, err := reg.Resolve(tt.req); err != nil || e.Name != tt.want {
				t.Fatal(e, err)
			}
		})
	}

	delete(reg.Editors, "4.8")
	if e, _ := reg.Resolve(Requirement{Version: "4.8-stable"}); e.Name != "a" {
		t.Fatal("linked by name order", e)
	}
	if _, err := reg.Resolve(Requirement{Version: "4.9-stable"}); !errors.Is(err, ErrNoEditor) || !strings.Contains(err.Error(), "gd editor install 4.9-stable") {
		t.Fatal("custom must not match by version", err)
	}
	reg.Default = "cust"
	if _, err := reg.Resolve(Requirement{Version: "4.9-stable"}); !errors.Is(err, ErrNoEditor) {
		t.Fatal("custom default must not match by version", err)
	}
}

func TestResolveLocalNameMissing(t *testing.T) {
	reg := &Registry{Editors: map[string]Editor{}}
	if _, err := reg.Resolve(Requirement{Local: "x"}); !errors.Is(err, ErrUnknownEditor) {
		t.Fatal(err)
	}
}

func TestResolveNoRequirementUsesDefault(t *testing.T) {
	reg := &Registry{Editors: map[string]Editor{}}
	if _, err := reg.Resolve(Requirement{}); !errors.Is(err, ErrNoDefault) {
		t.Fatal(err)
	}
	reg.Default = "gone"
	if _, err := reg.Resolve(Requirement{}); !errors.Is(err, ErrUnknownEditor) {
		t.Fatal(err)
	}
}

func TestVersionWarning(t *testing.T) {
	if VersionWarning("4.6", "4.6.1-stable") != "" || VersionWarning("", "4.6") != "" {
		t.Fatal("unexpected warning")
	}
	if w := VersionWarning("4.6", "4.7.2"); !strings.Contains(w, "4.6") || !strings.Contains(w, "4.7") {
		t.Fatal(w)
	}
}
