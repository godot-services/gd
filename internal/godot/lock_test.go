package godot

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLockRoundTrip(t *testing.T) {
	dir := t.TempDir()
	lock := Lock{
		"dialogic": {Source: "library:1234", URL: "https://x/a.zip", SHA256: "abc", VersionString: "2.0", Paths: []string{"addons/dialogic"}},
		"tool":     {Source: "git:https://github.com/a/b@v1", URL: "https://github.com/a/b/archive/v1.zip", SHA256: "def", Paths: []string{"addons/a", "addons/b"}},
	}
	if err := lock.Save(dir); err != nil {
		t.Fatal(err)
	}

	got, err := LoadLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, lock) {
		t.Fatalf("got %+v\nwant %+v", got, lock)
	}
}

func TestLoadLockMissingFile(t *testing.T) {
	lock, err := LoadLock(t.TempDir())
	if err != nil || len(lock) != 0 || lock == nil {
		t.Fatal(lock, err)
	}
}

func TestLockSaveEmptyRemovesFile(t *testing.T) {
	dir := t.TempDir()
	Lock{"a": {Source: "library:1", URL: "u", SHA256: "s", Paths: []string{"addons/a"}}}.Save(dir)

	if err := (Lock{}).Save(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, LockFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}
