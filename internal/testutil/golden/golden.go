// Package golden implements golden-file testing: a test's output is compared
// against a file checked into testdata/. When the output changes on purpose,
// regenerate the files with:
//
//	go test ./... -update
//
// and review the diff in git before committing.
package golden

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files in testdata/")

// Assert compares got with testdata/<name>.golden, or rewrites that file
// when the -update flag is set.
func Assert(t testing.TB, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("golden: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden: %v (run `go test ./... -update` to create it)", err)
	}
	if got != string(want) {
		t.Errorf("output differs from %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}
