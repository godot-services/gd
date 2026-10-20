package cmd

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/godot-services/gd/internal/editor"
	"github.com/godot-services/gd/internal/godot"
	"github.com/godot-services/gd/internal/godotbuilds"
	"github.com/godot-services/gd/internal/testutil/fakegodot"
)

func runEditor(t *testing.T, builds string, args ...string) (string, error) {
	t.Helper()
	return runOut(t, newFakeLibrary(t), append([]string{"--builds-url", builds}, args...)...)
}

func loadReg(t *testing.T) (*editor.Registry, string) {
	t.Helper()
	dir, err := editor.DataDir()
	if err != nil {
		t.Fatal(err)
	}
	r, err := editor.LoadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	return r, dir
}

func TestEditorInstallFromPackage(t *testing.T) {
	dir := newTestProject(t)
	os.WriteFile(filepath.Join(dir, "package.godot"), []byte("[editor]\nversion=4.7.2\n"), 0o644)
	out, err := runEditor(t, newFakeBuilds(t), "editor", "install")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := loadReg(t)
	e := r.Editors["4.7.2"]
	if e.Version != "4.7.2-stable" || e.Dotnet || r.Default != "4.7.2" || !exists(e.Path) {
		t.Errorf("registry = %+v default %q", e, r.Default)
	}
	if !strings.Contains(out, "installed 4.7.2 (4.7.2-stable) -> "+e.Path) {
		t.Errorf("out = %q", out)
	}
}

func TestEditorInstallNoVersion(t *testing.T) {
	newTestProject(t)
	if _, err := runEditor(t, newFakeBuilds(t), "editor", "install"); err == nil {
		t.Fatal("expected error")
	}
}

func TestEditorInstallDotnetKeepsDefault(t *testing.T) {
	newTestProject(t)
	b := newFakeBuilds(t)
	for _, a := range [][]string{{"4.7.2"}, {"4.7.2", "--dotnet"}} {
		if _, err := runEditor(t, b, append([]string{"editor", "install"}, a...)...); err != nil {
			t.Fatal(err)
		}
	}
	r, _ := loadReg(t)
	if !r.Editors["4.7.2-dotnet"].Dotnet || r.Default != "4.7.2" {
		t.Errorf("registry = %+v", r)
	}
}

func TestEditorInstallAlready(t *testing.T) {
	newTestProject(t)
	b := newFakeBuilds(t)
	runEditor(t, b, "editor", "install", "4.7.2")
	out, err := runEditor(t, b, "editor", "install", "4.7.2")
	if err != nil || !strings.Contains(out, "already installed") {
		t.Errorf("out %q err %v", out, err)
	}
}

func TestEditorUninstall(t *testing.T) {
	newTestProject(t)
	b := newFakeBuilds(t)
	runEditor(t, b, "editor", "install", "4.7.2")
	if _, err := runEditor(t, b, "editor", "uninstall", "4.7.2"); err != nil {
		t.Fatal(err)
	}
	r, dir := loadReg(t)
	if len(r.Editors) != 0 || r.Default != "" || exists(filepath.Join(dir, "editors", "4.7.2")) {
		t.Errorf("not removed: %+v", r)
	}
	if _, err := runEditor(t, b, "editor", "uninstall", "4.7.2"); err == nil {
		t.Error("unknown should fail")
	}
}

func TestEditorUninstallLinked(t *testing.T) {
	newTestProject(t)
	r, dir := loadReg(t)
	keep := filepath.Join(t.TempDir(), "godot")
	os.WriteFile(keep, nil, 0o755)
	r.Editors["mine"] = editor.Editor{Name: "mine", Path: keep, Source: "linked", Build: "custom"}
	r.Save(dir)
	_, err := runEditor(t, "", "editor", "uninstall", "mine")
	if err == nil || !strings.Contains(err.Error(), "gd editor unlink mine") || !exists(keep) {
		t.Errorf("err %v", err)
	}
}

func TestEditorList(t *testing.T) {
	dir := newTestProject(t)
	out, _ := runEditor(t, "", "editor", "list")
	if !strings.Contains(out, "no editors installed, run gd editor install <version>") {
		t.Errorf("out = %q", out)
	}
	b := newFakeBuilds(t)
	runEditor(t, b, "editor", "install", "4.7.2")
	runEditor(t, b, "editor", "install", "4.8", "--dotnet")
	os.WriteFile(filepath.Join(dir, "package.godot"), []byte("[editor]\nversion=4.8\ndotnet=true\n"), 0o644)
	out, err := runEditor(t, b, "editor", "list")
	if err != nil {
		t.Fatal(err)
	}
	var def, proj string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "4.7.2-stable") {
			def = l
		}
		if strings.Contains(l, "4.8-dotnet") {
			proj = l
		}
	}
	if !strings.HasPrefix(def, "*") || !strings.HasPrefix(proj, ">") || !strings.Contains(out, "NAME") {
		t.Errorf("out = %q", out)
	}
}

func TestEditorAvailable(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			w.Write([]byte(`[{"tag_name":"4.8-beta1"},{"tag_name":"4.7.2-stable"},{"tag_name":"3.6-stable"}]`))
			return
		}
		w.Write([]byte(`[]`))
	}))
	defer api.Close()
	out, err := runEditor(t, "", "--builds-api-url", api.URL, "editor", "available")
	if err != nil || out != "4.7.2-stable\n" {
		t.Errorf("out = %q, err = %v", out, err)
	}
	out, err = runEditor(t, "", "--builds-api-url", api.URL, "editor", "available", "--pre")
	if err != nil || out != "4.8-beta1\n4.7.2-stable\n" {
		t.Errorf("--pre out = %q, err = %v", out, err)
	}
}

func TestEditorAvailableNone(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`[]`)) }))
	defer api.Close()
	out, err := runEditor(t, "", "--builds-api-url", api.URL, "editor", "available")
	if err != nil || out != "no Godot 4 releases found\n" {
		t.Errorf("out = %q, err = %v", out, err)
	}
}

func linkEditor(t *testing.T, alias string) (string, error) {
	t.Helper()
	return runEditor(t, "", "editor", "link", alias, os.Args[0])
}

func TestEditorLinkOfficial(t *testing.T) {
	newTestProject(t)
	fakegodot.Setup(t, "4.7.2.stable.official.abc")
	out, err := linkEditor(t, "steam")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := loadReg(t)
	e := r.Editors["steam"]
	if e.Source != "linked" || e.Version != "4.7.2-stable" || e.Build != "official" || e.Path != os.Args[0] {
		t.Errorf("entry = %+v", e)
	}
	if !strings.Contains(out, "linked steam (4.7.2-stable, official) -> "+os.Args[0]) {
		t.Errorf("out = %q", out)
	}
}

func TestEditorLinkCustom(t *testing.T) {
	newTestProject(t)
	fakegodot.Setup(t, "4.7.2.stable.custom_build.abc")
	if _, err := linkEditor(t, "mine"); err != nil {
		t.Fatal(err)
	}
	if r, _ := loadReg(t); r.Editors["mine"].Build != "custom" {
		t.Errorf("entry = %+v", r.Editors["mine"])
	}
}

func TestEditorLinkRejects(t *testing.T) {
	newTestProject(t)
	fakegodot.Setup(t, "4.7.2.stable.official.abc")
	for _, a := range []string{"Bad Alias", "aliases", ""} {
		if _, err := linkEditor(t, a); err == nil {
			t.Errorf("alias %q accepted", a)
		}
	}
	if _, err := linkEditor(t, "one"); err != nil {
		t.Fatal(err)
	}
	if _, err := linkEditor(t, "one"); err == nil {
		t.Error("duplicate accepted")
	}
	if _, err := runEditor(t, "", "editor", "link", "gone", filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("missing path accepted")
	}
}

func TestEditorLinkProbeFailure(t *testing.T) {
	newTestProject(t)
	fakegodot.Setup(t, "4.7.2.stable.official.abc")
	t.Setenv("GD_FAKE_GODOT_EXIT", "3")
	if _, err := linkEditor(t, "broken"); err == nil {
		t.Fatal("expected error")
	}
	if r, _ := loadReg(t); len(r.Editors) != 0 {
		t.Errorf("registry = %+v", r)
	}
}

func TestEditorUnlink(t *testing.T) {
	newTestProject(t)
	fakegodot.Setup(t, "4.7.2.stable.official.abc")
	linkEditor(t, "steam")
	runEditor(t, newFakeBuilds(t), "editor", "install", "4.7.2")
	runEditor(t, "", "editor", "default", "steam")
	if _, err := runEditor(t, "", "editor", "unlink", "4.7.2"); err == nil || !strings.Contains(err.Error(), "gd editor uninstall 4.7.2") {
		t.Errorf("managed unlink err = %v", err)
	}
	if _, err := runEditor(t, "", "editor", "unlink", "nope"); !errors.Is(err, editor.ErrUnknownEditor) {
		t.Errorf("unknown err = %v", err)
	}
	if _, err := runEditor(t, "", "editor", "unlink", "steam"); err != nil {
		t.Fatal(err)
	}
	r, _ := loadReg(t)
	if _, ok := r.Editors["steam"]; ok || r.Default != "" || !exists(os.Args[0]) {
		t.Errorf("registry = %+v", r)
	}
}

func TestEditorDefault(t *testing.T) {
	newTestProject(t)
	fakegodot.Setup(t, "4.7.2.stable.official.abc")
	linkEditor(t, "steam")
	if _, err := runEditor(t, "", "editor", "default", "nope"); !errors.Is(err, editor.ErrUnknownEditor) {
		t.Errorf("err = %v", err)
	}
	out, err := runEditor(t, "", "editor", "default", "steam")
	if r, _ := loadReg(t); err != nil || r.Default != "steam" || !strings.Contains(out, "default editor: steam") {
		t.Errorf("default %q out %q err %v", r.Default, out, err)
	}
}

func TestEditorUse(t *testing.T) {
	dir := newTestProject(t)
	fakegodot.Setup(t, "4.7.2.stable.official.abc")
	linkEditor(t, "steam")
	local := filepath.Join(dir, "package.local.godot")
	if _, err := runEditor(t, "", "editor", "use", "nope"); !errors.Is(err, editor.ErrUnknownEditor) {
		t.Errorf("err = %v", err)
	}
	if _, err := runEditor(t, "", "editor", "use", "steam", "--clear"); err == nil {
		t.Error("name with --clear accepted")
	}
	if _, err := runEditor(t, "", "editor", "use", "steam"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(local); !strings.Contains(string(b), "steam") {
		t.Errorf("local file = %q", b)
	}
	if _, err := runEditor(t, "", "editor", "use", "--clear"); err != nil || exists(local) {
		t.Errorf("clear err %v exists %v", err, exists(local))
	}
}

func TestEditorUseGitignoreHint(t *testing.T) {
	dir := newTestProject(t)
	fakegodot.Setup(t, "4.7.2.stable.official.abc")
	linkEditor(t, "steam")
	hint := func() string {
		var errOut bytes.Buffer
		root := newRootCmd()
		root.SetArgs([]string{"editor", "use", "steam"})
		root.SetOut(io.Discard)
		root.SetErr(&errOut)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		return errOut.String()
	}
	if h := hint(); h != "" {
		t.Errorf("hint without .gitignore: %q", h)
	}
	gi := filepath.Join(dir, ".gitignore")
	os.WriteFile(gi, []byte(".godot/\n"), 0o644)
	if h := hint(); !strings.Contains(h, "hint: add package.local.godot to .gitignore") {
		t.Errorf("hint = %q", h)
	}
	os.WriteFile(gi, []byte(".godot/\n/package.local.godot\n"), 0o644)
	if h := hint(); h != "" {
		t.Errorf("hint with entry: %q", h)
	}
}

// orphanDir creates editors/<name> without a registry entry, holding the fake godot
// in this OS's layout when withExe is set.
func orphanDir(t *testing.T, name string, withExe bool) string {
	t.Helper()
	dd, err := editor.DataDir()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(dd, "editors", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if !withExe {
		os.WriteFile(filepath.Join(dir, "junk"), nil, 0o644)
		return dir
	}
	exe := map[string]string{"darwin": "Godot.app/Contents/MacOS/Godot", "windows": "Godot_v4.7.2-stable_win64.exe"}[runtime.GOOS]
	if exe == "" {
		exe = "Godot_v4.7.2-stable_linux.x86_64"
	}
	bin, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, filepath.FromSlash(exe))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, bin, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestEditorInstallRecoversOrphan(t *testing.T) {
	newTestProject(t)
	fakegodot.Setup(t, "4.7.2.stable.official.abc")
	dir := orphanDir(t, "4.7.2", true)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; http.NotFound(w, r) }))
	defer srv.Close()
	out, err := runEditor(t, srv.URL, "editor", "install", "4.7.2")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := loadReg(t)
	e := r.Editors["4.7.2"]
	if e.Source != "managed" || e.Version != "4.7.2-stable" || !strings.HasPrefix(e.Path, dir) || r.Default != "4.7.2" || hits != 0 {
		t.Errorf("entry = %+v default %q hits %d", e, r.Default, hits)
	}
	if !strings.Contains(out, "re-registered 4.7.2") {
		t.Errorf("out = %q", out)
	}
}

func TestEditorInstallReplacesBrokenOrphan(t *testing.T) {
	newTestProject(t)
	dir := orphanDir(t, "4.7.2", false)
	out, err := runEditor(t, newFakeBuilds(t), "editor", "install", "4.7.2")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := loadReg(t)
	if e := r.Editors["4.7.2"]; !exists(e.Path) || exists(filepath.Join(dir, "junk")) || !strings.Contains(out, "installed 4.7.2 (4.7.2-stable)") {
		t.Errorf("entry = %+v out %q", e, out)
	}
}

func TestEditorInstallStableSuffixSameName(t *testing.T) {
	newTestProject(t)
	b := newFakeBuilds(t)
	if _, err := runEditor(t, b, "editor", "install", "4.7.2-stable"); err != nil {
		t.Fatal(err)
	}
	out, err := runEditor(t, b, "editor", "install", "4.7.2")
	r, _ := loadReg(t)
	if err != nil || !strings.Contains(out, "already installed") || len(r.Editors) != 1 || r.Editors["4.7.2"].Name != "4.7.2" {
		t.Errorf("out %q err %v registry %+v", out, err, r.Editors)
	}
}

func TestInstallUnknownVersionHint(t *testing.T) {
	newTestProject(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	for _, c := range []string{"editor", "templates"} {
		_, err := runEditor(t, srv.URL, c, "install", "4.99")
		if !errors.Is(err, godotbuilds.ErrUnknownVersion) || !strings.Contains(err.Error(), "run gd editor available --pre to list versions") {
			t.Errorf("%s: err = %v", c, err)
		}
	}
}

func TestInstallReportsInvalidPackage(t *testing.T) {
	dir := newTestProject(t)
	os.WriteFile(filepath.Join(dir, "package.godot"), []byte("[editor]\nversion=3.x\n"), 0o644)
	for _, c := range []string{"editor", "templates"} {
		_, err := runEditor(t, newFakeBuilds(t), c, "install")
		if !errors.Is(err, godot.ErrInvalidGodotVersion) || strings.Contains(err.Error(), "no [editor] version") {
			t.Errorf("%s: err = %v", c, err)
		}
	}
}

func TestEditorUninstallRejectsBadName(t *testing.T) {
	newTestProject(t)
	dd, err := editor.DataDir()
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(dd, 0o755)
	keep := filepath.Join(dd, "keep")
	os.WriteFile(keep, nil, 0o644)
	os.WriteFile(filepath.Join(dd, editor.RegistryFileName), []byte("[..]\npath=/x\nversion=4.7.2-stable\nbuild=official\nsource=managed\n"), 0o644)
	if r, _ := loadReg(t); r.Editors[".."].Source != "managed" {
		t.Fatalf("registry = %+v", r.Editors)
	}
	if _, err := runEditor(t, "", "editor", "uninstall", ".."); err == nil || !exists(keep) {
		t.Errorf("err %v, data dir kept %v", err, exists(keep))
	}
}
