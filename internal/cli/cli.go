// Package cli parses command-line arguments and dispatches txoko's commands.
//
// Run takes its arguments and output streams as parameters instead of
// reaching for os.Args and os.Stdout, so the whole CLI can be exercised from
// unit tests.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/LifeOfAitor/txoko/internal/progress"
	"github.com/LifeOfAitor/txoko/internal/store"
	"github.com/LifeOfAitor/txoko/internal/tui"
	"github.com/LifeOfAitor/txoko/internal/version"
)

// Exit codes follow common Unix conventions.
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

const usageTemplate = `txoko — a cozy corner for your terminal that grows as you work.

Usage:
  txoko [flags]            open your scene and grow it by hand with +/-
  txoko --work [flags]     let your real activity grow it instead
  txoko <command>

Commands (for --work):
  hook install             record every commit (sets a global git hook)
  hook uninstall           remove the hook again
  hook status              show whether the hook is installed
  import [dir...]          import your recent commits from repositories
  stats                    print your activity summary
  log commit               record HEAD (used by the git hook)

Flags:
      --scene S    scene to show: %s (default %s)
      --seed N     scene layout; the default is unique to this machine
      --growth N   starting growth, 0 to 1 (default 0.4)
      --work       grow from your recorded activity (commits)
  -h, --help       show this help
  -v, --version    print version information

Keys:
  +/-  grow or shrink (not with --work)   s  next scene
  n  day / night / auto                   r  new layout
  space  pause                            q  quit
`

// usage is the help text with the scene names filled in.
var usage = fmt.Sprintf(usageTemplate, strings.Join(tui.SceneNames(), ", "), tui.SceneNames()[0])

// Dependencies that tests replace.
var (
	startTUI   = tui.Run
	executable = os.Executable
	workDir    = os.Getwd
	now        = time.Now
)

// Run executes txoko with the given arguments (without the program name)
// and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, rest := args[0], args[1:]
		switch cmd {
		case "log":
			return runLog(rest, stdout, stderr)
		case "hook":
			return runHook(rest, stdout, stderr)
		case "import":
			return runImport(rest, stdout, stderr)
		case "stats":
			return runStats(rest, stdout, stderr)
		case "help":
			fmt.Fprint(stdout, usage)
			return ExitOK
		}
		return usageError(stderr, fmt.Sprintf("unknown command %q", cmd))
	}
	return runScene(args, stdout, stderr)
}

// runScene opens the interactive scene.
func runScene(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("txoko")
	var (
		showVersion, work bool
		opts              = tui.Options{Growth: tui.DefaultGrowth}
	)
	fs.BoolVar(&showVersion, "version", false, "")
	fs.BoolVar(&showVersion, "v", false, "")
	fs.BoolVar(&work, "work", false, "")
	fs.Float64Var(&opts.Growth, "growth", tui.DefaultGrowth, "")
	fs.Uint64Var(&opts.Seed, "seed", 0, "")
	fs.StringVar(&opts.Scene, "scene", "", "")

	if code, done := parse(fs, args, stdout, stderr); done {
		return code
	}
	if fs.NArg() > 0 {
		return usageError(stderr, fmt.Sprintf("unknown command %q", fs.Arg(0)))
	}
	if showVersion {
		fmt.Fprint(stdout, version.Get())
		return ExitOK
	}
	if opts.Growth < 0 || opts.Growth > 1 {
		return usageError(stderr, fmt.Sprintf("--growth must be between 0 and 1, got %v", opts.Growth))
	}
	if opts.Scene != "" && !slices.Contains(tui.SceneNames(), opts.Scene) {
		return usageError(stderr, fmt.Sprintf("unknown scene %q (available: %s)",
			opts.Scene, strings.Join(tui.SceneNames(), ", ")))
	}

	if work && flagSet(fs, "growth") {
		return usageError(stderr, "--growth sets the scene by hand, so it can't be combined with --work")
	}

	// By default txoko is a visual toy you shape yourself; --work hands the
	// growth over to your recorded activity.
	if work {
		st, err := openStore()
		if err != nil {
			return fail(stderr, err)
		}
		defer func() { _ = st.Close() }()
		opts.Source = func() (progress.Summary, error) { return summarize(context.Background(), st) }
	}

	if err := startTUI(opts); err != nil {
		return fail(stderr, err)
	}
	return ExitOK
}

// summarize reads the recent events and computes what the scene shows.
func summarize(ctx context.Context, st *store.Store) (progress.Summary, error) {
	t := now()
	events, err := st.Since(ctx, t.Add(-progress.Window))
	if err != nil {
		return progress.Summary{}, err
	}
	return progress.Compute(events, t), nil
}

func openStore() (*store.Store, error) {
	path, err := store.DefaultPath()
	if err != nil {
		return nil, err
	}
	return store.Open(context.Background(), path)
}

// newFlagSet returns a flag set whose errors and help we print ourselves.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	return fs
}

// parse parses flags. done reports that the command already finished (help
// was shown or the flags were invalid) with the given exit code.
func parse(fs *flag.FlagSet, args []string, stdout, stderr io.Writer) (code int, done bool) {
	err := fs.Parse(args)
	switch {
	case err == nil:
		return 0, false
	case errors.Is(err, flag.ErrHelp):
		fmt.Fprint(stdout, usage)
		return ExitOK, true
	default:
		return usageError(stderr, err.Error()), true
	}
}

func flagSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) { found = found || f.Name == name })
	return found
}

func usageError(stderr io.Writer, msg string) int {
	fmt.Fprintf(stderr, "txoko: %s\n\n%s", msg, usage)
	return ExitUsage
}

func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "txoko: %v\n", err)
	return ExitError
}
