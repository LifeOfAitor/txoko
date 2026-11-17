package cli

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestOnceFlag(t *testing.T) {
	calls := fakeTUI(t, nil)
	code, stdout, stderr := run("--once", "--seed", "3")
	if code != ExitOK || len(*calls) != 0 {
		t.Fatalf("--once should print and exit without the live UI: exit %d %s", code, stderr)
	}
	if !strings.Contains(stdout, "txoko · bonsai") || !strings.Contains(stdout, "100%") {
		t.Errorf("--once output:\n%s", stdout)
	}
	if strings.Contains(stdout, "\x1b[") {
		t.Error("output to a non-terminal must be plain")
	}
}

func TestOnceGrowthAndWork(t *testing.T) {
	freshDB(t)
	if _, stdout, _ := run("--once", "--growth", "0.3"); !strings.Contains(stdout, " 30%") {
		t.Errorf("--growth should apply to --once:\n%s", stdout)
	}
	_, stdout, _ := run("--once", "--work")
	if !strings.Contains(stdout, "streak") || !strings.Contains(stdout, " 0%") {
		t.Errorf("--once --work should show activity:\n%s", stdout)
	}
}

func TestOnceSize(t *testing.T) {
	_, stdout, _ := run("--once", "--size", "20x8")
	if n := strings.Count(stdout, "\n"); n != 8 {
		t.Errorf("--size 20x8 should print 8 lines, got %d", n)
	}
	for _, tt := range []struct{ args []string }{
		{[]string{"--once", "--size", "big"}},
		{[]string{"--once", "--size", "5x5"}},
		{[]string{"--once", "--size", "40x"}},
		{[]string{"--size", "40x16"}},
		{[]string{"--once", "--pomodoro", "25"}},
	} {
		fakeTUI(t, nil)
		if code, _, _ := run(tt.args...); code != ExitUsage {
			t.Errorf("%v: exit %d, want usage error", tt.args, code)
		}
	}
}

func TestOnceWorkFailsWithoutStore(t *testing.T) {
	t.Setenv("TXOKO_DB", "/dev/null/txoko.db")
	if code, _, _ := run("--once", "--work"); code != ExitError {
		t.Errorf("exit %d, want %d", code, ExitError)
	}
}

func TestColorOutput(t *testing.T) {
	if colorOutput(io.Discard) {
		t.Error("a non-file writer is never colored")
	}
	orig := colorOutput
	colorOutput = func(io.Writer) bool { return true }
	t.Cleanup(func() { colorOutput = orig })
	if _, stdout, _ := run("--once"); !strings.Contains(stdout, "\x1b[") {
		t.Error("a terminal should get colors")
	}
}

func TestSceneSizeString(t *testing.T) {
	s := sceneSize{40, 16}
	if s.String() != "40x16" {
		t.Errorf("String() = %q", s.String())
	}
	_ = time.Now
}
