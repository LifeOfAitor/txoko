package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/LifeOfAitor/txoko/internal/tui"
)

// fakeTUI replaces the real UI for the duration of a test and records the
// options it was started with.
func fakeTUI(t *testing.T, err error) *[]tui.Options {
	t.Helper()
	var calls []tui.Options
	orig := startTUI
	startTUI = func(o tui.Options) error {
		calls = append(calls, o)
		return err
	}
	t.Cleanup(func() { startTUI = orig })
	return &calls
}

func run(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = Run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string // substring expected in stdout ("" = stdout must be empty)
		wantStderr string // substring expected in stderr ("" = stderr must be empty)
		wantTUI    bool
	}{
		{"no args starts the TUI", nil, ExitOK, "", "", true},
		{"long version flag", []string{"--version"}, ExitOK, "txoko ", "", false},
		{"short version flag", []string{"-v"}, ExitOK, "txoko ", "", false},
		{"long help flag", []string{"--help"}, ExitOK, "Usage:", "", false},
		{"short help flag", []string{"-h"}, ExitOK, "Usage:", "", false},
		{"unknown flag", []string{"--nope"}, ExitUsage, "", "flag provided but not defined", false},
		{"unknown command", []string{"bloom"}, ExitUsage, "", `unknown command "bloom"`, false},
		{"growth too high", []string{"--growth", "1.5"}, ExitUsage, "", "--growth must be between 0 and 1", false},
		{"growth negative", []string{"--growth=-0.1"}, ExitUsage, "", "--growth must be between 0 and 1", false},
		{"growth not a number", []string{"--growth", "lots"}, ExitUsage, "", "invalid value", false},
		{"known scene", []string{"--scene", "garden"}, ExitOK, "", "", true},
		{"unknown scene", []string{"--scene", "volcano"}, ExitUsage, "", `unknown scene "volcano" (available: bonsai, garden)`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := fakeTUI(t, nil)
			code, stdout, stderr := run(tt.args...)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			checkStream(t, "stdout", stdout, tt.wantStdout)
			checkStream(t, "stderr", stderr, tt.wantStderr)
			if started := len(*calls) > 0; started != tt.wantTUI {
				t.Errorf("TUI started = %v, want %v", started, tt.wantTUI)
			}
		})
	}
}

func TestRunPassesOptionsToTUI(t *testing.T) {
	calls := fakeTUI(t, nil)
	run("--seed", "7", "--growth", "0.9", "--scene", "garden")

	if len(*calls) != 1 {
		t.Fatalf("TUI started %d times, want 1", len(*calls))
	}
	if got := (*calls)[0]; got.Seed != 7 || got.Growth != 0.9 || got.Scene != "garden" {
		t.Errorf("options = %+v, want seed 7, growth 0.9 and the garden", got)
	}
}

func TestRunDefaultsGrowth(t *testing.T) {
	calls := fakeTUI(t, nil)
	run()
	if got := (*calls)[0].Growth; got != tui.DefaultGrowth {
		t.Errorf("growth = %v, want %v", got, tui.DefaultGrowth)
	}
}

func TestRunReportsTUIErrors(t *testing.T) {
	fakeTUI(t, errors.New("no terminal here"))
	code, _, stderr := run()
	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
	if !strings.Contains(stderr, "txoko: no terminal here") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestUsageListsScenes(t *testing.T) {
	_, stdout, _ := run("--help")
	if !strings.Contains(stdout, "bonsai, garden (default bonsai)") {
		t.Errorf("help should list scenes, got:\n%s", stdout)
	}
}

func TestRunErrorsIncludeUsage(t *testing.T) {
	_, _, stderr := run("--nope")
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("stderr should include usage on error, got:\n%s", stderr)
	}
}

func checkStream(t *testing.T, name, got, want string) {
	t.Helper()
	if want == "" {
		if got != "" {
			t.Errorf("%s should be empty, got:\n%s", name, got)
		}
		return
	}
	if !strings.Contains(got, want) {
		t.Errorf("%s = %q, want it to contain %q", name, got, want)
	}
}
