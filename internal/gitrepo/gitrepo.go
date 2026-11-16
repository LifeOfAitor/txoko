// Package gitrepo reads the few facts txoko needs from a git repository by
// running the git command itself, the same way you would in a shell.
package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Commit is a commit as txoko sees it.
type Commit struct {
	Hash string
	At   time.Time
}

// Errors callers may want to recognize.
var (
	ErrNotARepo = errors.New("not inside a git repository")
	ErrNoEmail  = errors.New("git user.email is not set, so your commits can't be told apart")
)

// git runs a git command in dir and returns its trimmed output.
func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	// Git translates its messages; force English so errors are recognizable
	// whatever the user's language (git said "no es un repositorio git" here).
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANGUAGE=C")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "not a git repository") {
			return "", ErrNotARepo
		}
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Name returns the repository's name: the folder name of its top level.
// Only the name is ever stored, never the full path.
func Name(ctx context.Context, dir string) (string, error) {
	top, err := git(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return filepath.Base(top), nil
}

// Head returns the commit HEAD points to.
func Head(ctx context.Context, dir string) (Commit, error) {
	out, err := git(ctx, dir, "log", "-1", "--format=%H %ct")
	if err != nil {
		return Commit{}, err
	}
	commits, err := parseLog(out)
	if err != nil || len(commits) == 0 {
		return Commit{}, fmt.Errorf("reading HEAD: unexpected output %q", out)
	}
	return commits[0], nil
}

// AuthorCommits returns commits by the configured user (user.email) since
// the given time, newest first. It is used to import past activity.
func AuthorCommits(ctx context.Context, dir string, since time.Time) ([]Commit, error) {
	email, err := git(ctx, dir, "config", "user.email")
	if err != nil || email == "" {
		return nil, ErrNoEmail
	}
	out, err := git(ctx, dir, "log", "--all", "--no-merges",
		"--author="+email, "--since="+since.Format(time.RFC3339), "--format=%H %ct")
	if err != nil {
		return nil, err
	}
	return parseLog(out)
}

// parseLog parses lines of "<hash> <unix time>".
func parseLog(out string) ([]Commit, error) {
	var commits []Commit
	for line := range strings.Lines(out) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		hash, ts, ok := strings.Cut(line, " ")
		if !ok {
			return nil, fmt.Errorf("unexpected git log line %q", line)
		}
		sec, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unexpected git log line %q", line)
		}
		commits = append(commits, Commit{Hash: hash, At: time.Unix(sec, 0)})
	}
	return commits, nil
}
