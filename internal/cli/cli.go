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
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"

	"github.com/LifeOfAitor/txoko/internal/notify"
	"github.com/LifeOfAitor/txoko/internal/progress"
	"github.com/LifeOfAitor/txoko/internal/scene"
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
  txoko --pomodoro 50      focus for 50 minutes while a new tree grows
  txoko --once [--work]    print it once, neofetch-style, and exit
  txoko <command>

Commands:
  config                   show where the settings file is
  config init              create it, with every setting explained
  hook install             record every commit (sets a global git hook)
  hook uninstall           remove the hook again
  hook status              show whether the hook is installed
  import [dir...]          import your recent commits from repositories
  daemon install           grow with your time at the computer, too (starts
                           a small background service when you log in)
  daemon uninstall         stop it again
  daemon status            show whether it's running, and today's active time
  stats                    print your activity summary
  status [--format F]      one line for tmux or your prompt, e.g. "🌳 72%% · 5d"
  snap [-o FILE] [flags]   save the scene as an SVG picture (default
                           txoko.svg; -o - writes it to stdout). Takes
                           --scene, --season, --sky, --growth, --work,
                           --seed and --size (default 60x22)
  log commit               record HEAD (used by the git hook)

Flags:
      --scene S    scene to show: %s (default %s)
      --seed N     scene layout; the default is unique to this machine
      --season S   spring, summer, autumn or winter (default: today's)
      --sky S      day or night (default: follows the clock)
      --growth N   starting growth, 0 to 1 (default 0.4)
      --work       grow from your recorded activity (commits and focus)
      --pomodoro M focus session of M minutes: the tree blooms as it ends
      --break M    rest M minutes after the session (with --pomodoro)
      --once       print the scene once with a little info panel, then exit
                   (neofetch-style, for your .zshrc); full grown unless
                   --growth or --work say otherwise
      --size WxH   scene size for --once (default 34x14)
  -h, --help       show this help
  -v, --version    print version information

Keys:
  +/-  grow or shrink (not with --work)   s  next scene   e  season
  n  day or night                         r  new layout
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
	notifySend = notify.Send
	lookPath   = exec.LookPath
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
		case "status":
			return runStatus(rest, stdout, stderr)
		case "config":
			return runConfig(rest, stdout, stderr)
		case "snap":
			return runSnap(rest, stdout, stderr)
		case "daemon":
			return runDaemon(rest, stdout, stderr)
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
	cfg, err := loadConfig()
	if err != nil {
		return fail(stderr, err)
	}
	// The config file provides the defaults; flags override them.
	fs := newFlagSet("txoko")
	var (
		showVersion, work bool
		opts              = tui.Options{Growth: tui.DefaultGrowth}
	)
	fs.BoolVar(&showVersion, "version", false, "")
	fs.BoolVar(&showVersion, "v", false, "")
	fs.BoolVar(&work, "work", cfg.Work, "")
	fs.Float64Var(&opts.Growth, "growth", tui.DefaultGrowth, "")
	fs.Uint64Var(&opts.Seed, "seed", cfg.Seed, "")
	fs.StringVar(&opts.Scene, "scene", cfg.Scene, "")
	fs.StringVar(&opts.Season, "season", cfg.Season, "")
	fs.StringVar(&opts.Sky, "sky", "", "")
	focus := minutes{lo: 1, hi: 240}
	brk := minutes{lo: 0, hi: 120, n: cfg.Pomodoro.Break}
	fs.Var(&focus, "pomodoro", "")
	fs.Var(&brk, "break", "")
	var once bool
	size := sceneSize{w: tui.OnceWidth, h: tui.OnceHeight}
	if cfg.Once.Size != "" {
		_ = size.Set(cfg.Once.Size) // already validated by loadConfig
	}
	fs.BoolVar(&once, "once", false, "")
	fs.Var(&size, "size", "")

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
	if msg := checkLook(opts); msg != "" {
		return usageError(stderr, msg)
	}
	// Asking for a hand-set growth or a pomodoro on the command line beats
	// `work = true` in the config file.
	if !flagSet(fs, "work") && (flagSet(fs, "growth") || flagSet(fs, "pomodoro")) {
		work = false
	}
	if once && !flagSet(fs, "work") && !flagSet(fs, "growth") {
		work = cfg.Once.Work
	}
	if work && flagSet(fs, "growth") {
		return usageError(stderr, "--growth sets the scene by hand, so it can't be combined with --work")
	}
	if msg := checkPomodoro(fs, work); msg != "" {
		return usageError(stderr, msg)
	}
	if once {
		if flagSet(fs, "pomodoro") {
			return usageError(stderr, "--once prints a single frame, so it can't run a --pomodoro")
		}
		return runOnce(opts, work, flagSet(fs, "growth"), size, stdout, stderr)
	}
	if flagSet(fs, "size") {
		return usageError(stderr, "--size only applies to --once")
	}
	if focus.n > 0 {
		opts.Focus = time.Duration(focus.n) * time.Minute
		opts.Break = time.Duration(brk.n) * time.Minute
		opts.OnFocusDone = recordFocus
		opts.OnBreakDone = func() {
			_ = notifySend("txoko", "Break's over — press enter to plant another tree.")
		}
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

// runOnce prints the scene a single time. Without --growth or --work the
// tree is shown fully grown: a greeting should look its best.
func runOnce(opts tui.Options, work, growthSet bool, size sceneSize, stdout, stderr io.Writer) int {
	o := tui.OnceOptions{
		Scene: opts.Scene, Season: opts.Season, Sky: opts.Sky, Seed: opts.Seed,
		Growth: 1, Width: size.w, Height: size.h,
		Color: colorOutput(stdout), Now: now,
	}
	if growthSet {
		o.Growth = opts.Growth
	}
	if work {
		st, err := openStore()
		if err != nil {
			return fail(stderr, err)
		}
		defer func() { _ = st.Close() }()
		s, err := summarize(context.Background(), st)
		if err != nil {
			return fail(stderr, err)
		}
		o.Summary, o.Growth = &s, s.Growth
	}
	if err := tui.Once(stdout, o); err != nil {
		return fail(stderr, err)
	}
	return ExitOK
}

// colorOutput reports whether w should get colors: only when it is a
// terminal and the user hasn't asked for no color (https://no-color.org).
var colorOutput = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(f.Fd()) && os.Getenv("NO_COLOR") == ""
}

// sceneSize is a flag value like "40x16".
type sceneSize struct{ w, h int }

func (s *sceneSize) String() string { return fmt.Sprintf("%dx%d", s.w, s.h) }

func (s *sceneSize) Set(v string) error {
	ws, hs, ok := strings.Cut(strings.ToLower(strings.TrimSpace(v)), "x")
	w, errW := strconv.Atoi(ws)
	h, errH := strconv.Atoi(hs)
	if !ok || errW != nil || errH != nil {
		return fmt.Errorf("use WIDTHxHEIGHT, like 40x16")
	}
	if w < 10 || w > 300 || h < 5 || h > 120 {
		return fmt.Errorf("use a width of 10 to 300 and a height of 5 to 120")
	}
	s.w, s.h = w, h
	return nil
}

// minutes is a flag value holding a whole number of minutes within a range.
// The flag package calls Set while parsing, so bad input such as "0.5",
// "-3" or "abc" is rejected right there, with a message that says what to
// type instead.
type minutes struct {
	n      int
	lo, hi int
}

func (m *minutes) String() string { return strconv.Itoa(m.n) }

func (m *minutes) Set(s string) error {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fmt.Errorf("use whole minutes, like 25")
	}
	if n < m.lo || n > m.hi {
		return fmt.Errorf("use between %d and %d minutes", m.lo, m.hi)
	}
	m.n = n
	return nil
}

// checkPomodoro validates how the pomodoro flags combine with others. It returns a usage message,
// or "" when they are fine.
func checkPomodoro(fs *flag.FlagSet, work bool) string {
	switch {
	case !flagSet(fs, "pomodoro"):
		if flagSet(fs, "break") {
			return "--break only works together with --pomodoro"
		}
		return ""
	case work:
		return "--pomodoro grows its own tree, so it can't be combined with --work"
	case flagSet(fs, "growth"):
		return "--pomodoro starts from a seed, so it can't be combined with --growth"
	}
	return ""
}

// recordFocus saves a completed focus session, so it also grows the --work
// scene, and lets you know it's done.
func recordFocus(length time.Duration) error {
	minutes := int(length / time.Minute)
	_ = notifySend("txoko", fmt.Sprintf("%s of focus — your tree is in full bloom 🌳", plural(minutes, "minute")))

	st, err := openStore()
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	at := now()
	_, err = st.Add(context.Background(), store.Event{
		Kind:   store.KindFocus,
		Repo:   "pomodoro",
		Ref:    fmt.Sprintf("focus-%d", at.UnixNano()),
		At:     at,
		Amount: float64(minutes),
	})
	return err
}

// checkLook checks the flags that pick what a scene looks like, and says
// what's wrong, if anything.
func checkLook(opts tui.Options) string {
	if opts.Growth < 0 || opts.Growth > 1 {
		return fmt.Sprintf("--growth must be between 0 and 1, got %v", opts.Growth)
	}
	if opts.Scene != "" && !slices.Contains(tui.SceneNames(), opts.Scene) {
		return fmt.Sprintf("unknown scene %q (available: %s)", opts.Scene, strings.Join(tui.SceneNames(), ", "))
	}
	if _, ok := scene.ParseSeason(opts.Season); opts.Season != "" && !ok {
		return fmt.Sprintf("unknown season %q (use spring, summer, autumn or winter)", opts.Season)
	}
	if opts.Sky != "" && opts.Sky != "day" && opts.Sky != "night" {
		return fmt.Sprintf("--sky takes day or night, got %q", opts.Sky)
	}
	return ""
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
