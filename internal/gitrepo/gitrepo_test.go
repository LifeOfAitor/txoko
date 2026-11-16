package gitrepo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// isolate keeps git from reading the developer's real configuration.
func isolate(t *testing.T) {
	t.Helper()
	cfg := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(cfg, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// newRepo creates a throwaway repository named "garden" with an author set.
func newRepo(t *testing.T) string {
	t.Helper()
	isolate(t)
	dir := filepath.Join(t.TempDir(), "garden")
	run(t, "", "init", "-q", dir)
	run(t, dir, "config", "user.email", "me@example.com")
	run(t, dir, "config", "user.name", "Me")
	return dir
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func commitAt(t *testing.T, dir, msg string, at time.Time) {
	t.Helper()
	t.Setenv("GIT_AUTHOR_DATE", at.Format(time.RFC3339))
	t.Setenv("GIT_COMMITTER_DATE", at.Format(time.RFC3339))
	run(t, dir, "commit", "-q", "--allow-empty", "-m", msg)
}

func TestNameAndHead(t *testing.T) {
	ctx := context.Background()
	dir := newRepo(t)
	at := time.Date(2026, 11, 16, 18, 30, 0, 0, time.UTC)
	commitAt(t, dir, "first", at)

	sub := filepath.Join(dir, "deep", "inside")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	name, err := Name(ctx, sub)
	if err != nil || name != "garden" {
		t.Errorf("Name = %q, %v; want garden", name, err)
	}

	head, err := Head(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(head.Hash) != 40 || !head.At.Equal(at) {
		t.Errorf("Head = %+v", head)
	}
}

func TestAuthorCommits(t *testing.T) {
	ctx := context.Background()
	dir := newRepo(t)
	now := time.Now()
	commitAt(t, dir, "old", now.Add(-60*24*time.Hour))
	commitAt(t, dir, "recent", now.Add(-2*24*time.Hour))
	commitAt(t, dir, "today", now.Add(-time.Hour))

	// Someone else's commit must not count.
	t.Setenv("GIT_AUTHOR_EMAIL", "other@example.com")
	commitAt(t, dir, "theirs", now.Add(-30*time.Minute))
	t.Setenv("GIT_AUTHOR_EMAIL", "")
	_ = os.Unsetenv("GIT_AUTHOR_EMAIL")

	got, err := AuthorCommits(ctx, dir, now.Add(-30*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d commits, want 2 (recent and today): %+v", len(got), got)
	}
}

func TestAuthorCommitsNeedsEmail(t *testing.T) {
	isolate(t)
	dir := filepath.Join(t.TempDir(), "r")
	run(t, "", "init", "-q", dir)
	if _, err := AuthorCommits(context.Background(), dir, time.Now()); err == nil {
		t.Error("expected an error without user.email")
	}
}

func TestNotARepo(t *testing.T) {
	isolate(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	dir := t.TempDir()
	if _, err := Name(context.Background(), dir); !errors.Is(err, ErrNotARepo) {
		t.Errorf("Name error = %v, want ErrNotARepo", err)
	}
	if _, err := Head(context.Background(), dir); !errors.Is(err, ErrNotARepo) {
		t.Errorf("Head error = %v, want ErrNotARepo", err)
	}
}

func TestHeadWithoutCommits(t *testing.T) {
	dir := newRepo(t)
	if _, err := Head(context.Background(), dir); err == nil {
		t.Error("Head of an empty repository should fail")
	}
}

func TestParseLog(t *testing.T) {
	got, err := parseLog("abc 100\n\ndef 200\n")
	if err != nil || len(got) != 2 || got[1].Hash != "def" || got[1].At.Unix() != 200 {
		t.Errorf("parseLog = %+v, %v", got, err)
	}
	for _, bad := range []string{"nospace", "abc notanumber"} {
		if _, err := parseLog(bad); err == nil {
			t.Errorf("parseLog(%q) should fail", bad)
		}
	}
}
