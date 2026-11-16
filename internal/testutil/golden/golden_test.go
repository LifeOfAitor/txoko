package golden

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssertMatches(t *testing.T) {
	t.Chdir(t.TempDir())
	writeGolden(t, "ok", "hello\n")
	Assert(t, "ok", "hello\n")
}

func TestAssertUpdateWritesFile(t *testing.T) {
	t.Chdir(t.TempDir())
	*update = true
	t.Cleanup(func() { *update = false })

	Assert(t, "nested/new", "fresh\n")

	got, err := os.ReadFile(filepath.Join("testdata", "nested", "new.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fresh\n" {
		t.Errorf("written = %q, want %q", got, "fresh\n")
	}
}

func TestAssertReportsMismatchAndMissing(t *testing.T) {
	t.Chdir(t.TempDir())
	writeGolden(t, "diff", "want\n")

	for _, tc := range []struct{ name, file, got string }{
		{"mismatch", "diff", "got\n"},
		{"missing file", "absent", "anything"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recorder{TB: t}
			func() {
				defer func() { _ = recover() }() // Fatalf on the recorder panics to stop like the real one
				Assert(rec, tc.file, tc.got)
			}()
			if !rec.failed {
				t.Error("Assert should have failed")
			}
		})
	}
}

// recorder captures failures instead of failing the real test.
type recorder struct {
	testing.TB
	failed bool
}

func (r *recorder) Helper()               {}
func (r *recorder) Errorf(string, ...any) { r.failed = true }
func (r *recorder) Fatalf(string, ...any) { r.failed = true; panic("fatal") }

func writeGolden(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("testdata", name+".golden"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
