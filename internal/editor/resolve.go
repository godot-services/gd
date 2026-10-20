package editor

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/godot-services/gd/internal/godot"
)

var (
	ErrNoEditor      = errors.New("no matching Godot editor")
	ErrUnknownEditor = errors.New("unknown editor")
	ErrNoDefault     = errors.New("no default editor")
)

// Requirement is what a project asks of its editor; Version is a release tag, "" = none.
type Requirement struct {
	Version string
	Dotnet  bool
	Local   string // package.local.godot name
}

// Resolve picks the project's editor: local name, default, managed, then linked by name.
// Custom builds are only reachable via Local or the default alias, never by version.
func (r *Registry) Resolve(req Requirement) (Editor, error) {
	if req.Local != "" {
		e, ok := r.Editors[req.Local]
		if !ok {
			return Editor{}, fmt.Errorf("%w %q, run gd editor use --clear or gd editor default", ErrUnknownEditor, req.Local)
		}
		return e, nil
	}
	d, hasDefault := r.Editors[r.Default]
	if req.Version == "" {
		if r.Default == "" {
			return Editor{}, fmt.Errorf("%w, run gd editor default <name>", ErrNoDefault)
		}
		if !hasDefault {
			return Editor{}, fmt.Errorf("%w %q, run gd editor default <name>", ErrUnknownEditor, r.Default)
		}
		return d, nil
	}
	match := func(e Editor) bool {
		return e.Build == "official" && e.Version == req.Version && e.Dotnet == req.Dotnet
	}
	if hasDefault && match(d) {
		return d, nil
	}
	for _, source := range []string{"managed", "linked"} {
		for _, name := range slices.Sorted(maps.Keys(r.Editors)) {
			if e := r.Editors[name]; e.Source == source && match(e) {
				return e, nil
			}
		}
	}
	return Editor{}, fmt.Errorf("%w for Godot %s, run gd editor install %s", ErrNoEditor, req.Version, req.Version)
}

// VersionWarning is non-empty when the major.minor of both versions differ.
func VersionWarning(projectVersion, editorVersion string) string {
	p, e := godot.MajorMinor(projectVersion), godot.MajorMinor(editorVersion)
	if p == "" || e == "" || p == e {
		return ""
	}
	return fmt.Sprintf("warning: project.godot targets Godot %s but the editor is %s", p, e)
}
