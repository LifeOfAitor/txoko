package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LifeOfAitor/txoko/internal/store"
)

func TestSnapSavesAPicture(t *testing.T) {
	freshDB(t)
	path := filepath.Join(t.TempDir(), "arch.svg")
	code, out, errOut := run("snap", "-o", path, "--scene", "arch", "--season", "winter", "--sky", "night", "--seed", "3")
	if code != ExitOK || errOut != "" || !strings.Contains(out, "saved "+path+" (60×22)") {
		t.Fatalf("code %d, stdout %q, stderr %q", code, out, errOut)
	}
	svg, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(svg), "<svg") || !strings.Contains(string(svg), "txoko · arch · winter · night") {
		t.Errorf("not the picture asked for:\n%.300s", svg)
	}
}

func TestSnapDefaultsAndStdout(t *testing.T) {
	freshDB(t)
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if code, out, _ := run("snap"); code != ExitOK || !strings.Contains(out, "saved txoko.svg") {
		t.Errorf("default: code %d, %q", code, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "txoko.svg")); err != nil {
		t.Errorf("txoko.svg should be saved here: %v", err)
	}
	code, out, _ := run("snap", "-o", "-", "--size", "30x10")
	if code != ExitOK || !strings.HasPrefix(out, "<svg") || strings.Contains(out, "saved") || !strings.Contains(out, `viewBox="0 0 356 286"`) {
		t.Errorf("-o -: code %d, %.200q", code, out)
	}
}

func TestSnapWork(t *testing.T) {
	freshDB(t)
	st, err := openStore()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = st.Add(context.Background(), store.Event{Kind: store.KindCommit, Repo: "r", Ref: "a", At: now()})
	_ = st.Close()
	_, grown, _ := run("snap", "-o", "-", "--seed", "1", "--sky", "day")
	_, work, _ := run("snap", "-o", "-", "--seed", "1", "--sky", "day", "--work")
	if grown == work {
		t.Error("--work grows the scene from your activity, not fully")
	}
}

func TestSnapErrors(t *testing.T) {
	freshDB(t)
	for _, args := range [][]string{
		{"snap", "extra"},
		{"snap", "--scene", "nope"},
		{"snap", "--growth", "2"},
		{"snap", "--season", "monsoon"},
		{"snap", "--sky", "dusk"},
		{"snap", "--work", "--growth", "0.5"},
		{"snap", "--size", "big"},
	} {
		if code, _, _ := run(args...); code != ExitUsage {
			t.Errorf("%v: code %d, want %d", args, code, ExitUsage)
		}
	}
	if code, _, errOut := run("snap", "-o", filepath.Join(t.TempDir(), "missing", "x.svg")); code != ExitError || !strings.Contains(errOut, "saving the picture") {
		t.Errorf("unwritable path: code %d, %q", code, errOut)
	}
	if code, out, _ := run("snap", "--help"); code != ExitOK || !strings.Contains(out, "snap [-o FILE]") {
		t.Errorf("--help: code %d", code)
	}
}
