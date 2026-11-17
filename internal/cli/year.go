package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/x/term"

	"github.com/LifeOfAitor/txoko/internal/contrib"
	"github.com/LifeOfAitor/txoko/internal/tui"
)

// Where txoko reads contributions from. Tests point them at a fake GitHub.
var (
	githubAPI   = contrib.Endpoint
	githubToken = contrib.EnvToken
)

// runYear sums up your last year of contributions: from GitHub, or from
// txoko's own record with --local.
func runYear(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("year")
	user := fs.String("user", "", "GitHub user to show (default: you)")
	local := fs.Bool("local", false, "use txoko's own record instead of GitHub")
	refresh := fs.Bool("refresh", false, "ask GitHub again even if it was asked recently")
	sky := fs.String("sky", "", "day or night")
	size := sceneSize{w: tui.YearWidth, h: tui.YearHeight}
	fs.Var(&size, "size", "landscape size, e.g. 100x14")
	if code, done := parse(fs, args, stdout, stderr); done {
		return code
	}
	if fs.NArg() > 0 {
		return usageError(stderr, "usage: txoko year [--user LOGIN] [--local] [--refresh] [--sky S] [--size WxH]")
	}
	if *sky != "" && *sky != "day" && *sky != "night" {
		return usageError(stderr, fmt.Sprintf("--sky must be day or night, not %q", *sky))
	}
	if !flagSet(fs, "size") {
		if width, ok := terminalWidth(stdout); ok {
			size.w = width
		}
	}
	if *local && (*user != "" || *refresh) {
		return usageError(stderr, "--local reads txoko's own record: it can't take --user or --refresh")
	}

	st, err := openStore()
	if err != nil {
		return fail(stderr, err)
	}
	defer func() { _ = st.Close() }()

	ctx := context.Background()
	t := now()
	var cal contrib.Calendar
	if *local {
		events, err := st.Since(ctx, t.AddDate(-1, 0, 0))
		if err != nil {
			return fail(stderr, err)
		}
		times := make([]time.Time, len(events))
		for i, e := range events {
			times[i] = e.At
		}
		cal = contrib.FromTimes(times, t.AddDate(-1, 0, 1), t, time.Local)
		cal.Source, cal.Fetched = "txoko's own record", t
	} else {
		key := "github:@me"
		if *user != "" {
			key = "github:" + *user
		}
		var warning error
		cal, warning, err = contrib.Cached(ctx, st, key, t, *refresh, func(ctx context.Context) (contrib.Calendar, error) {
			token, err := githubToken()
			if err != nil {
				return contrib.Calendar{}, err
			}
			return contrib.GitHub{Endpoint: githubAPI, Token: token}.Calendar(ctx, *user)
		})
		if err != nil {
			fmt.Fprintf(stderr, "txoko: %v\n(txoko year --local shows your year from txoko's own record instead)\n", err)
			return ExitError
		}
		if warning != nil {
			fmt.Fprintf(stderr, "txoko: couldn't update from GitHub (%v); showing what it said %s\n", warning, tui.Ago(t.Sub(cal.Fetched)))
		}
	}
	err = tui.Year(stdout, tui.YearOptions{
		Calendar: cal, Width: size.w, Height: size.h, Sky: *sky,
		Color: colorOutput(stdout), Now: now,
	})
	if err != nil {
		return fail(stderr, err)
	}
	return ExitOK
}

// terminalWidth is the width of the terminal w writes to, if it is one.
var terminalWidth = func(w io.Writer) (int, bool) {
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(f.Fd()) {
		return 0, false
	}
	width, _, err := term.GetSize(f.Fd())
	return width, err == nil && width > 0
}
