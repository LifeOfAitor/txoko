package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/LifeOfAitor/txoko/internal/daemon"
	"github.com/LifeOfAitor/txoko/internal/idle"
	"github.com/LifeOfAitor/txoko/internal/progress"
)

// What the daemon commands reach for outside txoko. Tests replace them,
// so they never install a real service or wait a real minute.
var (
	serviceRun  = serviceRunDefault
	goos        = runtime.GOOS
	homeDir     = os.UserHomeDir
	idleReader  = idle.System
	trackEvery  = daemon.Every
	daemonUntil = daemonUntilDefault
)

// serviceRunDefault runs launchctl or systemctl.
func serviceRunDefault(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

// daemonUntilDefault lasts until the service is told to stop.
func daemonUntilDefault() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// runDaemon handles `txoko daemon install|uninstall|status|run`.
func runDaemon(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return usageError(stderr, "usage: txoko daemon install|uninstall|status|run")
	}
	switch args[0] {
	case "install":
		return daemonInstall(stdout, stderr)
	case "uninstall":
		svc, err := service("")
		if err != nil {
			return fail(stderr, err)
		}
		if err := svc.Uninstall(); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintln(stdout, "✓ active time off — your recorded activity is kept")
		return ExitOK
	case "status":
		return daemonStatus(stdout, stderr)
	case "run":
		return daemonRun(stderr)
	}
	return usageError(stderr, fmt.Sprintf("unknown daemon command %q", args[0]))
}

func service(exe string) (daemon.Service, error) {
	home, err := homeDir()
	if err != nil {
		return daemon.Service{}, fmt.Errorf("finding home directory: %w", err)
	}
	return daemon.Service{GOOS: goos, Home: home, Exe: exe, UID: os.Getuid(), Run: serviceRun}, nil
}

func daemonInstall(stdout, stderr io.Writer) int {
	// Make sure this system can tell when you're at the computer before
	// leaving a service behind that can't.
	if _, err := idleReader()(); err != nil {
		return fail(stderr, err)
	}
	exe, err := executable()
	if err != nil {
		return fail(stderr, err)
	}
	exe = stableExecutable(exe)
	if strings.Contains(exe, "go-build") {
		return fail(stderr, errors.New("install the daemon from a built binary (make build, then ./bin/txoko daemon install), not from go run"))
	}
	svc, err := service(exe)
	if err != nil {
		return fail(stderr, err)
	}
	if err := svc.Install(); err != nil {
		return fail(stderr, err)
	}
	path, _ := svc.File()
	fmt.Fprintf(stdout, "✓ active time on — time at the computer now grows your scene too\n")
	fmt.Fprintf(stdout, "  runs:    %s daemon run, from when you log in\n", exe)
	fmt.Fprintf(stdout, "  records: only the minutes you're at the computer, nothing else\n")
	fmt.Fprintf(stdout, "  service: %s\n", path)
	fmt.Fprintf(stdout, "  undo it: txoko daemon uninstall\n")
	return ExitOK
}

func daemonStatus(stdout, stderr io.Writer) int {
	svc, err := service("")
	if err != nil {
		return fail(stderr, err)
	}
	st, err := svc.Check()
	if err != nil {
		return fail(stderr, err)
	}
	switch {
	case st.Running:
		today := "no activity yet today"
		if db, err := openStore(); err == nil {
			if s, err := summarize(context.Background(), db); err == nil && s.ActiveToday > 0 {
				today = progress.Clock(s.ActiveToday) + " at the computer today"
			}
			_ = db.Close()
		}
		fmt.Fprintf(stdout, "✓ running — %s\n", today)
	case st.Installed:
		where := "journalctl --user -u txoko"
		if goos == "darwin" {
			where = svc.LogFile()
		}
		fmt.Fprintf(stdout, "✗ installed but not running — see %s\n", where)
	default:
		fmt.Fprintln(stdout, "✗ not installed — run: txoko daemon install")
	}
	return ExitOK
}

// daemonRun is what the service runs: it records active minutes until it's
// told to stop.
func daemonRun(stderr io.Writer) int {
	read := idleReader()
	if _, err := read(); err != nil {
		return fail(stderr, err)
	}
	st, err := openStore()
	if err != nil {
		return fail(stderr, err)
	}
	defer func() { _ = st.Close() }()

	ctx, stop := daemonUntil()
	defer stop()
	daemon.Tracker{Idle: read, Record: st.AddActive, Now: now, Every: trackEvery}.Run(ctx, func(err error) {
		fmt.Fprintf(stderr, "%s txoko: %v\n", now().Format(time.DateTime), err)
	})
	return ExitOK
}
