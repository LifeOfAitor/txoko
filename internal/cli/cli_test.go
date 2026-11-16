package cli

import (
	"bytes"
	"strings"
	"testing"
)

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
	}{
		{"no args", nil, ExitOK, "still sprouting", ""},
		{"long version flag", []string{"--version"}, ExitOK, "txoko ", ""},
		{"short version flag", []string{"-v"}, ExitOK, "txoko ", ""},
		{"long help flag", []string{"--help"}, ExitOK, "Usage:", ""},
		{"short help flag", []string{"-h"}, ExitOK, "Usage:", ""},
		{"unknown flag", []string{"--nope"}, ExitUsage, "", "flag provided but not defined"},
		{"unknown command", []string{"bloom"}, ExitUsage, "", `unknown command "bloom"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := run(tt.args...)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			checkStream(t, "stdout", stdout, tt.wantStdout)
			checkStream(t, "stderr", stderr, tt.wantStderr)
		})
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
