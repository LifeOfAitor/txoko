// Package cli parses command-line arguments and dispatches txoko's commands.
//
// Run takes its arguments and output streams as parameters instead of
// reaching for os.Args and os.Stdout, so the whole CLI can be exercised from
// unit tests.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/LifeOfAitor/txoko/internal/tui"
	"github.com/LifeOfAitor/txoko/internal/version"
)

// Exit codes follow common Unix conventions.
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

const usage = `txoko — a cozy corner for your terminal that grows as you work.

Usage:
  txoko [flags]

Flags:
      --growth N   starting growth of the demo garden, 0 to 1 (default 0.4)
      --seed N     garden layout; the default is unique to this machine
  -h, --help       show this help
  -v, --version    print version information

Keys:
  +/-  grow or shrink   n  day / night / auto   r  new layout
  space  pause          q  quit
`

// startTUI launches the interactive UI. Tests replace it so they never need
// a real terminal.
var startTUI = tui.Run

// Run executes txoko with the given arguments (without the program name)
// and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("txoko", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // errors and usage are printed by us, not by flag
	fs.Usage = func() {}

	var (
		showVersion bool
		opts        = tui.Options{Growth: tui.DefaultGrowth}
	)
	fs.BoolVar(&showVersion, "version", false, "")
	fs.BoolVar(&showVersion, "v", false, "")
	fs.Float64Var(&opts.Growth, "growth", tui.DefaultGrowth, "")
	fs.Uint64Var(&opts.Seed, "seed", 0, "")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return ExitOK
		}
		return usageError(stderr, err.Error())
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

	if err := startTUI(opts); err != nil {
		fmt.Fprintf(stderr, "txoko: %v\n", err)
		return ExitError
	}
	return ExitOK
}

func usageError(stderr io.Writer, msg string) int {
	fmt.Fprintf(stderr, "txoko: %s\n\n%s", msg, usage)
	return ExitUsage
}
