package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/LifeOfAitor/txoko/internal/contrib"
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
	if code, done := parse(fs, args, stdout, stderr); done {
		return code
	}
	if fs.NArg() > 0 {
		return usageError(stderr, "usage: txoko year [--user LOGIN] [--local] [--refresh]")
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
			fmt.Fprintf(stderr, "txoko: couldn't update from GitHub (%v); showing what it said %s\n", warning, ago(t.Sub(cal.Fetched)))
		}
	}
	printYear(stdout, cal, t)
	return ExitOK
}

// printYear prints the summary of a calendar.
func printYear(w io.Writer, cal contrib.Calendar, t time.Time) {
	fmt.Fprintf(w, "txoko · your year · %s\n", cal.Source)
	fmt.Fprintf(w, "  total    %s in the last year\n", plural(cal.Total(), "contribution"))
	if best, ok := cal.BestWeek(); ok {
		fmt.Fprintf(w, "  best     %d in the week of %s\n", best.Count, best.Start.Format("2 Jan"))
	}
	fmt.Fprintf(w, "  streak   %s at most in a row\n", plural(cal.LongestStreak(), "day"))
	fmt.Fprintf(w, "  updated  %s\n", ago(t.Sub(cal.Fetched)))
}

// ago says how long ago something happened, roughly.
func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return plural(int(d/time.Minute), "minute") + " ago"
	case d < 48*time.Hour:
		return plural(int(d/time.Hour), "hour") + " ago"
	}
	return plural(int(d/(24*time.Hour)), "day") + " ago"
}
