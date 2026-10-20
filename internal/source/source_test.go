package source

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitArchiveURL(t *testing.T) {
	cases := []struct {
		repo, ref, want string
		err             error
	}{
		{"https://github.com/a/b", "v1", "https://github.com/a/b/archive/v1.zip", nil},
		{"https://github.com/a/b.git", "v1", "https://github.com/a/b/archive/v1.zip", nil},
		{"https://gitlab.com/a/b/", "main", "https://gitlab.com/a/b/-/archive/main.zip", nil},
		{"https://codeberg.org/a/b", "abc123", "https://codeberg.org/a/b/archive/abc123.zip", nil},
		{"https://example.com/a/b", "v1", "", ErrUnsupportedGitHost},
	}
	for _, c := range cases {
		got, err := GitArchiveURL(c.repo, c.ref)
		if !errors.Is(err, c.err) || got != c.want {
			t.Errorf("%s@%s: got %q, %v; want %q, %v", c.repo, c.ref, got, err, c.want, c.err)
		}
	}
}

func fakeLibrary(t *testing.T, status int, body string) Library {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/asset/1234" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Library{BaseURL: srv.URL, Client: srv.Client()}
}

func TestLibraryResolve(t *testing.T) {
	lib := fakeLibrary(t, 200, `{"title":"Dialogic 2","version_string":"2.0","download_url":"https://x/y.zip"}`)
	got, err := lib.Resolve(context.Background(), "1234")
	want := Resolved{URL: "https://x/y.zip", Title: "Dialogic 2", VersionString: "2.0", Source: "library:1234"}
	if err != nil || got != want {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestLibraryResolveNotFound(t *testing.T) {
	lib := fakeLibrary(t, 200, `{}`)
	if _, err := lib.Resolve(context.Background(), "999"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLibraryResolveNoDownloadURL(t *testing.T) {
	lib := fakeLibrary(t, 200, `{"title":"x","download_url":""}`)
	if _, err := lib.Resolve(context.Background(), "1234"); !errors.Is(err, ErrNoDownloadURL) {
		t.Fatal(err)
	}
}

func TestResolveGit(t *testing.T) {
	got, err := Resolve(context.Background(), Library{}, godotAsset("", "https://github.com/a/b", "v1"))
	want := Resolved{URL: "https://github.com/a/b/archive/v1.zip", Title: "b", VersionString: "v1", Source: "git:https://github.com/a/b@v1"}
	if err != nil || got != want {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{"Dialogic 2": "dialogic_2", "  Foo--Bar! ": "foo_bar"} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRepoName(t *testing.T) {
	if got := RepoName("https://github.com/a/my_tool.git/"); got != "my_tool" {
		t.Fatal(got)
	}
}
