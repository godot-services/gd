package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// Without SetOut, cobra's cmd.Printf goes to stderr; normal output must reach stdout.
func TestNormalOutputGoesToStdout(t *testing.T) {
	newTestProject(t)
	lib := newFakeLibrary(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = stdout })

	var errb bytes.Buffer
	root := newRootCmd()
	root.SetArgs([]string{"--library-url", lib.URL, "add", "1234"})
	root.SetErr(&errb)
	err = root.Execute()
	w.Close()
	os.Stdout = stdout
	out, _ := io.ReadAll(r)

	if err != nil || !strings.Contains(string(out), "added dialogic_2") || errb.Len() != 0 {
		t.Fatalf("err %v\nstdout %q\nstderr %q", err, out, errb.String())
	}
}
