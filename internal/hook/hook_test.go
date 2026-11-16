package hook

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// sandbox points git's global config at a temporary file, so tests never
// touch the developer's real ~/.gitconfig.
func sandbox(t *testing.T) (gitconfig string) {
	t.Helper()
	gitconfig = filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(gitconfig, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", gitconfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	return gitconfig
}

func gitRun(t *testing.T, dir string, args ...string) error {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(out))
	}
	return nil
}

// fakeTxoko writes a script that records its arguments to a log file.
func fakeTxoko(t *testing.T) (exe, logFile string) {
	t.Helper()
	dir := t.TempDir()
	logFile = filepath.Join(dir, "calls.log")
	exe = filepath.Join(dir, "fake txoko") // a space, to exercise quoting
	script := "#!/bin/sh\necho \"$@\" >> '" + logFile + "'\n"
	if err := os.WriteFile(exe, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe, logFile
}

func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "me@example.com"},
		{"config", "user.name", "Me"},
	} {
		if err := gitRun(t, dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestInstallRecordsCommits(t *testing.T) {
	ctx := context.Background()
	sandbox(t)
	hooks := filepath.Join(t.TempDir(), "hooks")
	exe, logFile := fakeTxoko(t)

	if err := Install(ctx, hooks, exe); err != nil {
		t.Fatal(err)
	}
	st, err := Check(ctx, hooks)
	if err != nil || !st.Installed || st.HooksPath != hooks {
		t.Fatalf("Check = %+v, %v", st, err)
	}

	repo := newRepo(t)
	if err := gitRun(t, repo, "commit", "-q", "--allow-empty", "-m", "hello"); err != nil {
		t.Fatal(err)
	}

	// The hook runs txoko in the background, so give it a moment.
	deadline := time.Now().Add(5 * time.Second)
	for {
		body, _ := os.ReadFile(logFile)
		if strings.Contains(string(body), "log commit") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("txoko was not called by the hook; log: %q", body)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestRepositoryHooksStillRun(t *testing.T) {
	ctx := context.Background()
	sandbox(t)
	hooks := filepath.Join(t.TempDir(), "hooks")
	exe, _ := fakeTxoko(t)
	if err := Install(ctx, hooks, exe); err != nil {
		t.Fatal(err)
	}

	repo := newRepo(t)
	// A repository pre-commit hook that rejects every commit.
	local := filepath.Join(repo, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(local, []byte("#!/bin/sh\necho blocked by repo hook >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := gitRun(t, repo, "commit", "-q", "--allow-empty", "-m", "nope")
	if err == nil || !strings.Contains(err.Error(), "blocked by repo hook") {
		t.Fatalf("the repository's own pre-commit hook must still run, got: %v", err)
	}
}

// TestHooksAreSilent: with the hooks installed, everyday git commands such
// as init, commit and clone must not print anything extra.
func TestHooksAreSilent(t *testing.T) {
	ctx := context.Background()
	sandbox(t)
	if err := Install(ctx, filepath.Join(t.TempDir(), "hooks"), "/nonexistent/txoko"); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "src")
	clone := filepath.Join(t.TempDir(), "clone")
	steps := [][]string{
		{"-c", "init.defaultBranch=main", "init", "-q", src},
		{"-C", src, "-c", "user.email=me@example.com", "-c", "user.name=Me", "commit", "-q", "--allow-empty", "-m", "x"},
		{"clone", "-q", src, clone},
	}
	for _, args := range steps {
		cmd := exec.Command("git", args...)
		out, err := cmd.CombinedOutput()
		if err != nil || len(out) > 0 {
			t.Errorf("git %v printed %q (err %v)", args, out, err)
		}
	}
}

func TestInstallRefusesForeignHooksPath(t *testing.T) {
	ctx := context.Background()
	sandbox(t)
	if err := gitRun(t, "", "config", "--global", "core.hooksPath", "/somewhere/else"); err != nil {
		t.Fatal(err)
	}
	err := Install(ctx, filepath.Join(t.TempDir(), "hooks"), "/bin/txoko")
	var taken ErrHooksPathTaken
	if !errors.As(err, &taken) || taken.Current != "/somewhere/else" {
		t.Fatalf("Install error = %v, want ErrHooksPathTaken", err)
	}
	if !strings.Contains(err.Error(), "/somewhere/else") {
		t.Errorf("error should name the current path: %v", err)
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	ctx := context.Background()
	sandbox(t)
	hooks := filepath.Join(t.TempDir(), "hooks")
	for range 2 {
		if err := Install(ctx, hooks, "/bin/txoko"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUninstall(t *testing.T) {
	ctx := context.Background()
	sandbox(t)
	hooks := filepath.Join(t.TempDir(), "hooks")
	if err := Install(ctx, hooks, "/bin/txoko"); err != nil {
		t.Fatal(err)
	}
	// A file the user put there must survive.
	mine := filepath.Join(hooks, "my-notes.txt")
	if err := os.WriteFile(mine, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Uninstall(ctx, hooks); err != nil {
		t.Fatal(err)
	}
	st, _ := Check(ctx, hooks)
	if st.Installed || st.HooksPath != "" {
		t.Errorf("after uninstall: %+v", st)
	}
	if _, err := os.Stat(mine); err != nil {
		t.Error("uninstall removed a file txoko did not write")
	}
	if _, err := os.Stat(filepath.Join(hooks, "post-commit")); !os.IsNotExist(err) {
		t.Error("post-commit hook should be gone")
	}
	// Uninstalling again, or with nothing installed, is fine.
	if err := Uninstall(ctx, filepath.Join(t.TempDir(), "never")); err != nil {
		t.Errorf("uninstall with nothing installed: %v", err)
	}
}

func TestUninstallLeavesForeignHooksPath(t *testing.T) {
	ctx := context.Background()
	sandbox(t)
	if err := gitRun(t, "", "config", "--global", "core.hooksPath", "/theirs"); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(ctx, filepath.Join(t.TempDir(), "hooks")); err != nil {
		t.Fatal(err)
	}
	if st, _ := Check(ctx, "x"); st.HooksPath != "/theirs" {
		t.Errorf("uninstall changed someone else's hooksPath to %q", st.HooksPath)
	}
}

func TestScript(t *testing.T) {
	post := Script("post-commit", "/Users/it's me/txoko")
	for _, want := range []string{marker, `'/Users/it'\''s me/txoko' log commit`, "hooks/post-commit", "exec"} {
		if !strings.Contains(post, want) {
			t.Errorf("post-commit script missing %q:\n%s", want, post)
		}
	}
	pre := Script("pre-push", "/bin/txoko")
	if strings.Contains(pre, "log commit") || !strings.Contains(pre, "hooks/pre-push") {
		t.Errorf("pre-push should only pass through:\n%s", pre)
	}
}

func TestDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/cfg")
	if d, _ := Dir(); d != "/cfg/txoko/hooks" {
		t.Errorf("Dir = %s", d)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/me")
	if d, _ := Dir(); d != "/home/me/.config/txoko/hooks" {
		t.Errorf("Dir fallback = %s", d)
	}
}
