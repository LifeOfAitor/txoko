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

	"github.com/LifeOfAitor/txoko/internal/version"
)

// Exit codes follow common Unix conventions.
const (
	ExitOK    = 0
	ExitUsage = 2
)

const usage = `txoko — a cozy corner for your terminal that grows as you work.

Usage:
  txoko [flags]

Flags:
  -h, --help       show this help
  -v, --version    print version information
`

// Run executes txoko with the given arguments (without the program name)
// and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("txoko", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // errors and usage are printed by us, not by flag
	fs.Usage = func() {}

	var showVersion bool
	fs.BoolVar(&showVersion, "version", false, "print version information")
	fs.BoolVar(&showVersion, "v", false, "print version information")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return ExitOK
		}
		fmt.Fprintf(stderr, "txoko: %v\n\n%s", err, usage)
		return ExitUsage
	}

	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "txoko: unknown command %q\n\n%s", fs.Arg(0), usage)
		return ExitUsage
	}

	if showVersion {
		fmt.Fprint(stdout, version.Get())
		return ExitOK
	}

	fmt.Fprintln(stdout, "🌱 txoko is still sprouting — nothing to show yet. Try `txoko --version`.")
	return ExitOK
}
