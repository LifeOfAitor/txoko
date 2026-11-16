package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LifeOfAitor/txoko/internal/progress"
	"github.com/LifeOfAitor/txoko/internal/tui"
)

// freshDB gives the test its own empty database.
func freshDB(t *testing.T) {
	t.Helper()
	t.Setenv("TXOKO_DB", filepath.Join(t.TempDir(), "txoko.db"))
}

// inDir makes the CLI believe it was started from dir.
func inDir(t *testing.T, dir string) {
	t.Helper()
	orig := workDir
	workDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { workDir = orig })
}

// newRepo creates a repository named "orchard" with commits by "me" at the
// given times.
func newRepo(t *testing.T, times ...time.Time) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "orchard")
	git(t, "", "init", "-q", dir)
	git(t, dir, "config", "user.email", "me@example.com")
	git(t, dir, "config", "user.name", "Me")
	for i, at := range times {
		t.Setenv("GIT_AUTHOR_DATE", at.Format(time.RFC3339))
		t.Setenv("GIT_COMMITTER_DATE", at.Format(time.RFC3339))
		git(t, dir, "commit", "-q", "--allow-empty", "-m", "c"+string(rune('0'+i)))
	}
	return dir
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestSceneUsesRealActivityByDefault(t *testing.T) {
	freshDB(t)
	var (
		got    progress.Summary
		err    error
		called bool
	)
	orig := startTUI
	startTUI = func(o tui.Options) error {
		if o.Source == nil {
			t.Fatal("the scene should be fed by the activity store")
		}
		called = true
		got, err = o.Source() // read while the UI is "open", like the real one
		return nil
	}
	t.Cleanup(func() { startTUI = orig })

	if code, _, stderr := run(); code != ExitOK || !called {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if err != nil || got != (progress.Summary{}) {
		t.Errorf("empty store: %+v, %v", got, err)
	}
}

func TestDemoModes(t *testing.T) {
	for _, args := range [][]string{{"--demo"}, {"--growth", "0.3"}} {
		calls := fakeTUI(t, nil)
		run(args...)
		if (*calls)[0].Source != nil {
			t.Errorf("%v should run the demo without a data source", args)
		}
	}
}

func TestSceneFailsWhenStoreCannotOpen(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TXOKO_DB", filepath.Join(blocker, "txoko.db"))
	fakeTUI(t, nil)
	if code, _, _ := run(); code != ExitError {
		t.Errorf("exit = %d, want %d", code, ExitError)
	}
}

func TestLogCommit(t *testing.T) {
	freshDB(t)
	inDir(t, newRepo(t, time.Now().Add(-time.Minute)))

	code, stdout, stderr := run("log", "commit")
	if code != ExitOK || !strings.Contains(stdout, "recorded") || !strings.Contains(stdout, "orchard") {
		t.Fatalf("first log: exit %d\n%s%s", code, stdout, stderr)
	}
	if _, stdout, _ := run("log", "commit"); !strings.Contains(stdout, "already recorded") {
		t.Errorf("second log should be a no-op, got %q", stdout)
	}
	if _, stdout, _ := run("stats"); !strings.Contains(stdout, "today   1 commit\n") {
		t.Errorf("stats after one commit:\n%s", stdout)
	}
}

func TestLogErrors(t *testing.T) {
	freshDB(t)
	if code, _, _ := run("log"); code != ExitUsage {
		t.Errorf("`log` without kind: exit %d", code)
	}
	if code, _, _ := run("log", "push"); code != ExitUsage {
		t.Errorf("`log push`: exit %d", code)
	}
	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	inDir(t, t.TempDir())
	if code, _, stderr := run("log", "commit"); code != ExitError || !strings.Contains(stderr, "not inside a git repository") {
		t.Errorf("outside a repo: exit %d, %q", code, stderr)
	}
}

func TestImport(t *testing.T) {
	freshDB(t)
	n := time.Now()
	repo := newRepo(t, n.Add(-90*24*time.Hour), n.Add(-3*24*time.Hour), n.Add(-time.Hour))

	code, stdout, stderr := run("import", repo)
	if code != ExitOK || !strings.Contains(stdout, "2 new of 2 commits") {
		t.Fatalf("import: exit %d\n%s%s", code, stdout, stderr)
	}
	if _, stdout, _ := run("import", repo); !strings.Contains(stdout, "0 new of 2 commits") {
		t.Errorf("re-import should add nothing:\n%s", stdout)
	}
	if _, stdout, _ := run("import", "--days", "120", repo); !strings.Contains(stdout, "1 new of 3 commits") {
		t.Errorf("a longer window should pick up the old commit:\n%s", stdout)
	}

	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	if code, _, stderr := run("import", t.TempDir()); code != ExitOK || !strings.Contains(stderr, "not a git repository") {
		t.Errorf("non-repo should be skipped: exit %d %q", code, stderr)
	}
}

func TestImportDefaultsToCurrentDir(t *testing.T) {
	freshDB(t)
	repo := newRepo(t, time.Now().Add(-time.Hour))
	t.Chdir(repo)
	if _, stdout, _ := run("import"); !strings.Contains(stdout, "1 new of 1 commits") {
		t.Errorf("import with no args should use the current directory:\n%s", stdout)
	}
}

func TestImportErrors(t *testing.T) {
	freshDB(t)
	if code, _, _ := run("import", "--days", "0"); code != ExitUsage {
		t.Errorf("--days 0: exit %d", code)
	}
	if code, _, _ := run("import", "--nope"); code != ExitUsage {
		t.Errorf("bad flag: exit %d", code)
	}
	if code, stdout, _ := run("import", "-h"); code != ExitOK || !strings.Contains(stdout, "Usage:") {
		t.Errorf("import -h: exit %d", code)
	}
	// A repository without user.email can't tell which commits are yours.
	dir := filepath.Join(t.TempDir(), "anon")
	git(t, "", "init", "-q", dir)
	if code, _, stderr := run("import", dir); code != ExitError || !strings.Contains(stderr, "user.email") {
		t.Errorf("repo without email: exit %d %q", code, stderr)
	}
}

func TestStats(t *testing.T) {
	freshDB(t)
	_, stdout, _ := run("stats")
	for _, want := range []string{"txoko · stats", "growth  ▱▱▱▱▱▱▱▱▱▱   0%", "today   0 commits", "streak  0 days", "recent  0 commits in the last 120 days"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stats missing %q:\n%s", want, stdout)
		}
	}
	if code, _, _ := run("stats", "extra"); code != ExitUsage {
		t.Errorf("stats with args: exit %d", code)
	}
}

func TestHookLifecycle(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	gitconfig := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(gitconfig, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", gitconfig)
	fakeExe(t, "/opt/txoko/bin/txoko")

	if _, stdout, _ := run("hook", "status"); !strings.Contains(stdout, "not installed") {
		t.Errorf("status before install:\n%s", stdout)
	}
	code, stdout, stderr := run("hook", "install")
	if code != ExitOK || !strings.Contains(stdout, "hook installed") || !strings.Contains(stdout, "/opt/txoko/bin/txoko") {
		t.Fatalf("install: exit %d\n%s%s", code, stdout, stderr)
	}
	if _, stdout, _ := run("hook", "status"); !strings.Contains(stdout, "✓ installed") {
		t.Errorf("status after install:\n%s", stdout)
	}
	if code, stdout, _ := run("hook", "uninstall"); code != ExitOK || !strings.Contains(stdout, "removed") {
		t.Errorf("uninstall: exit %d %q", code, stdout)
	}
	if _, stdout, _ := run("hook", "status"); !strings.Contains(stdout, "not installed") {
		t.Errorf("status after uninstall:\n%s", stdout)
	}

	git(t, "", "config", "--global", "core.hooksPath", "/husky")
	if _, stdout, _ := run("hook", "status"); !strings.Contains(stdout, "points to /husky") {
		t.Errorf("status with a foreign hooksPath:\n%s", stdout)
	}
	if code, _, stderr := run("hook", "install"); code != ExitError || !strings.Contains(stderr, "won't replace it") {
		t.Errorf("install over a foreign hooksPath: exit %d %q", code, stderr)
	}
}

func TestHookRefusesGoRun(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	fakeExe(t, "/var/folders/xy/go-build1234/b001/exe/txoko")
	if code, _, stderr := run("hook", "install"); code != ExitError || !strings.Contains(stderr, "not from go run") {
		t.Errorf("install from go run: exit %d %q", code, stderr)
	}
}

func TestHookUsage(t *testing.T) {
	for _, args := range [][]string{{"hook"}, {"hook", "explode"}, {"hook", "install", "now"}} {
		if code, _, _ := run(args...); code != ExitUsage {
			t.Errorf("%v: exit %d, want usage error", args, code)
		}
	}
}

func TestHelpCommand(t *testing.T) {
	code, stdout, _ := run("help")
	if code != ExitOK || !strings.Contains(stdout, "hook install") {
		t.Errorf("help: exit %d\n%s", code, stdout)
	}
}

func fakeExe(t *testing.T, path string) {
	t.Helper()
	orig := executable
	executable = func() (string, error) { return path, nil }
	t.Cleanup(func() { executable = orig })
}

func TestPlural(t *testing.T) {
	if plural(1, "day") != "1 day" || plural(0, "day") != "0 days" || plural(5, "commit") != "5 commits" {
		t.Error("plural is wrong")
	}
}
