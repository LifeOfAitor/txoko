package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/LifeOfAitor/txoko/internal/gitrepo"
	"github.com/LifeOfAitor/txoko/internal/hook"
	"github.com/LifeOfAitor/txoko/internal/progress"
	"github.com/LifeOfAitor/txoko/internal/store"
)

// runLog records an event. The git hook calls `txoko log commit` right
// after every commit, from inside the repository.
func runLog(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || args[0] != "commit" {
		return usageError(stderr, "usage: txoko log commit")
	}
	ctx := context.Background()
	dir, err := workDir()
	if err != nil {
		return fail(stderr, err)
	}
	name, err := gitrepo.Name(ctx, dir)
	if err != nil {
		return fail(stderr, err)
	}
	head, err := gitrepo.Head(ctx, dir)
	if err != nil {
		return fail(stderr, err)
	}

	st, err := openStore()
	if err != nil {
		return fail(stderr, err)
	}
	defer func() { _ = st.Close() }()
	added, err := st.Add(ctx, store.Event{Kind: store.KindCommit, Repo: name, Ref: head.Hash, At: head.At})
	if err != nil {
		return fail(stderr, err)
	}
	if added {
		fmt.Fprintf(stdout, "🌱 recorded %s in %s\n", head.Hash[:7], name)
	} else {
		fmt.Fprintf(stdout, "already recorded %s\n", head.Hash[:7])
	}
	return ExitOK
}

// runHook manages the global git hook.
func runHook(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return usageError(stderr, "usage: txoko hook install|uninstall|status")
	}
	ctx := context.Background()
	dir, err := hook.Dir()
	if err != nil {
		return fail(stderr, err)
	}

	switch args[0] {
	case "install":
		exe, err := executable()
		if err != nil {
			return fail(stderr, err)
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		// `go run` builds into a temporary folder that disappears afterwards,
		// which would leave the hook pointing at nothing.
		if strings.Contains(exe, "go-build") {
			return fail(stderr, errors.New("install the hook from a built binary (make build, then ./bin/txoko hook install), not from go run"))
		}
		if err := hook.Install(ctx, dir, exe); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintf(stdout, "✓ hook installed — every commit now grows your scene\n")
		fmt.Fprintf(stdout, "  hooks:   %s\n  runs:    %s\n", dir, exe)
		fmt.Fprintf(stdout, "  undo it: txoko hook uninstall\n")
		fmt.Fprintf(stdout, "  tip:     txoko import ~/code/*  brings in your recent commits\n")
		return ExitOK

	case "uninstall":
		if err := hook.Uninstall(ctx, dir); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintln(stdout, "✓ hook removed — your recorded activity is kept")
		return ExitOK

	case "status":
		st, err := hook.Check(ctx, dir)
		if err != nil {
			return fail(stderr, err)
		}
		switch {
		case st.Installed:
			fmt.Fprintf(stdout, "✓ installed (%s)\n", st.Dir)
		case st.HooksPath != "":
			fmt.Fprintf(stdout, "✗ not installed: core.hooksPath points to %s\n", st.HooksPath)
		default:
			fmt.Fprintln(stdout, "✗ not installed — run: txoko hook install")
		}
		return ExitOK
	}
	return usageError(stderr, fmt.Sprintf("unknown hook command %q", args[0]))
}

// runImport brings in the user's recent commits from existing repositories,
// so the scene doesn't start empty.
func runImport(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("import")
	days := fs.Int("days", 30, "")
	if code, done := parse(fs, args, stdout, stderr); done {
		return code
	}
	if *days <= 0 {
		return usageError(stderr, "--days must be positive")
	}
	dirs := fs.Args()
	if len(dirs) == 0 {
		dirs = []string{"."}
	}

	ctx := context.Background()
	st, err := openStore()
	if err != nil {
		return fail(stderr, err)
	}
	defer func() { _ = st.Close() }()

	since := now().AddDate(0, 0, -*days)
	code := ExitOK
	for _, dir := range dirs {
		name, err := gitrepo.Name(ctx, dir)
		if errors.Is(err, gitrepo.ErrNotARepo) {
			fmt.Fprintf(stderr, "  skip %s: not a git repository\n", dir)
			continue
		}
		if err != nil {
			fmt.Fprintf(stderr, "  skip %s: %v\n", dir, err)
			code = ExitError
			continue
		}
		commits, err := gitrepo.AuthorCommits(ctx, dir, since)
		if err != nil {
			fmt.Fprintf(stderr, "  skip %s: %v\n", name, err)
			code = ExitError
			continue
		}
		added := 0
		for _, c := range commits {
			ok, err := st.Add(ctx, store.Event{Kind: store.KindCommit, Repo: name, Ref: c.Hash, At: c.At})
			if err != nil {
				return fail(stderr, err)
			}
			if ok {
				added++
			}
		}
		fmt.Fprintf(stdout, "  %-20s %d new of %d commits\n", name, added, len(commits))
	}
	return code
}

// runStats prints the activity summary.
func runStats(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		return usageError(stderr, "usage: txoko stats")
	}
	st, err := openStore()
	if err != nil {
		return fail(stderr, err)
	}
	defer func() { _ = st.Close() }()
	s, err := summarize(context.Background(), st)
	if err != nil {
		return fail(stderr, err)
	}

	filled := int(s.Growth*10 + 0.5)
	fmt.Fprintf(stdout, "txoko · stats\n")
	fmt.Fprintf(stdout, "  growth  %s%s %3.0f%%\n", strings.Repeat("▰", filled), strings.Repeat("▱", 10-filled), s.Growth*100)
	fmt.Fprintf(stdout, "  today   %s, %d min of focus\n", plural(s.Today, "commit"), s.FocusToday)
	fmt.Fprintf(stdout, "  streak  %s\n", plural(s.Streak, "day"))
	fmt.Fprintf(stdout, "  recent  %s in the last %d days\n", plural(s.Recent, "commit"), int(progress.Window/(24*time.Hour)))
	return ExitOK
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
