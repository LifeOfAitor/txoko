// Package hook installs txoko's global git hooks.
//
// Git looks for hooks in the folder named by core.hooksPath. txoko points
// the global core.hooksPath at its own folder, where:
//
//   - post-commit records the new commit (in the background, so committing
//     never gets slower) and then runs the repository's own post-commit;
//   - every other hook is a tiny passthrough that runs the repository's own
//     hook, so repos that rely on .git/hooks keep working exactly as before.
//
// Installing refuses to overwrite a core.hooksPath set by someone else, and
// uninstalling only ever removes files txoko wrote.
package hook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// marker identifies files written by txoko.
const marker = "# txoko-managed-hook"

// passthrough lists the client-side hooks git may run. All of them get a
// passthrough so a global hooks folder doesn't hide a repository's own.
var passthrough = []string{
	"applypatch-msg", "pre-applypatch", "post-applypatch",
	"pre-commit", "pre-merge-commit", "prepare-commit-msg", "commit-msg",
	"post-commit", "pre-rebase", "post-checkout", "post-merge",
	"pre-push", "post-rewrite", "pre-auto-gc", "reference-transaction",
	"sendemail-validate", "post-index-change",
}

// ErrHooksPathTaken means core.hooksPath already points somewhere else.
type ErrHooksPathTaken struct{ Current string }

func (e ErrHooksPathTaken) Error() string {
	return fmt.Sprintf("git core.hooksPath is already set to %q; txoko won't replace it", e.Current)
}

// Status describes the current installation.
type Status struct {
	Installed bool   // core.hooksPath points at txoko's folder
	Dir       string // txoko's hooks folder
	HooksPath string // current global core.hooksPath ("" if unset)
}

// Dir returns txoko's hooks folder: $XDG_CONFIG_HOME/txoko/hooks, falling
// back to ~/.config/txoko/hooks.
func Dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "txoko", "hooks"), nil
}

// Check reports whether the hooks are installed.
func Check(ctx context.Context, dir string) (Status, error) {
	current, err := hooksPath(ctx)
	if err != nil {
		return Status{}, err
	}
	return Status{Installed: current == dir, Dir: dir, HooksPath: current}, nil
}

// Install writes the hooks into dir and points the global core.hooksPath at
// it. exe is the absolute path of the txoko binary the hook should run.
func Install(ctx context.Context, dir, exe string) error {
	current, err := hooksPath(ctx)
	if err != nil {
		return err
	}
	if current != "" && current != dir {
		return ErrHooksPathTaken{Current: current}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating hooks folder: %w", err)
	}
	for _, name := range passthrough {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(Script(name, exe)), 0o755); err != nil {
			return fmt.Errorf("writing %s hook: %w", name, err)
		}
	}
	_, err = git(ctx, "config", "--global", "core.hooksPath", dir)
	return err
}

// Uninstall removes txoko's hooks and unsets core.hooksPath if it points at
// them. Files that txoko did not write are left alone.
func Uninstall(ctx context.Context, dir string) error {
	current, err := hooksPath(ctx)
	if err != nil {
		return err
	}
	if current == dir {
		if _, err := git(ctx, "config", "--global", "--unset", "core.hooksPath"); err != nil {
			return err
		}
	}

	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if body, err := os.ReadFile(path); err == nil && bytes.Contains(body, []byte(marker)) {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}
	_ = os.Remove(dir) // only succeeds if nothing else is left
	return nil
}

// Script returns the hook script for the given hook name.
func Script(name, exe string) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\n" + marker + "\n")
	if name == "post-commit" {
		b.WriteString("# Records each commit so your txoko scene grows. Remove with: txoko hook uninstall\n")
		fmt.Fprintf(&b, "%s log commit >/dev/null 2>&1 &\n", shellQuote(exe))
	} else {
		fmt.Fprintf(&b, "# Runs this repository's own %s hook, if it has one.\n", name)
	}
	// Some hooks (reference-transaction) fire while `git init` or `git clone`
	// is still creating the repository: stay silent if it isn't ready yet.
	b.WriteString("gitdir=\"$(git rev-parse --git-common-dir 2>/dev/null)\" || exit 0\n")
	fmt.Fprintf(&b, "hook=\"$gitdir/hooks/%s\"\n", name)
	b.WriteString("if [ -x \"$hook\" ]; then exec \"$hook\" \"$@\"; fi\n")
	b.WriteString("exit 0\n")
	return b.String()
}

// shellQuote wraps s in single quotes, escaping any it contains.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func hooksPath(ctx context.Context) (string, error) {
	out, err := git(ctx, "config", "--global", "--get", "core.hooksPath")
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return "", nil // exit code 1 means "not set"
	}
	return out, err
}

func git(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANGUAGE=C")
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
