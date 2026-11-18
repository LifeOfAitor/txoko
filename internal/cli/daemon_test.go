package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LifeOfAitor/txoko/internal/daemon"
	"github.com/LifeOfAitor/txoko/internal/idle"
	"github.com/LifeOfAitor/txoko/internal/store"
)

// daemonTest sets up a fake macOS: a throwaway home, a txoko binary, an
// idle time and a launchctl that only writes down what it was asked.
// refuse makes the launchctl calls starting with it fail.
func daemonTest(t *testing.T, idleTime time.Duration, refuse ...string) *[]string {
	t.Helper()
	freshDB(t)
	home := t.TempDir()
	exe := filepath.Join(t.TempDir(), "txoko")
	if err := os.WriteFile(exe, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var ran []string
	set(t, &goos, "darwin")
	set(t, &homeDir, func() (string, error) { return home, nil })
	set(t, &executable, func() (string, error) { return exe, nil })
	set(t, &idleReader, func() idle.Reader { return func() (time.Duration, error) { return idleTime, nil } })
	set(t, &serviceRun, func(name string, args ...string) error {
		mu.Lock()
		defer mu.Unlock()
		cmd := name + " " + strings.Join(args, " ")
		ran = append(ran, cmd)
		for _, r := range refuse {
			if strings.HasPrefix(cmd, r) {
				return errors.New("refused")
			}
		}
		return nil
	})
	return &ran
}

// set replaces *v with x for the rest of the test.
func set[T any](t *testing.T, v *T, x T) {
	t.Helper()
	orig := *v
	*v = x
	t.Cleanup(func() { *v = orig })
}

func TestDaemonInstall(t *testing.T) {
	ran := daemonTest(t, 10*time.Second)
	code, out, errOut := run("daemon", "install")
	if code != ExitOK || errOut != "" {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"✓ active time on", "daemon run, from when you log in", "nothing else", "txoko daemon uninstall"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	home, _ := homeDir()
	if _, err := os.Stat(filepath.Join(home, "Library", "LaunchAgents", daemon.Label+".plist")); err != nil {
		t.Errorf("no plist: %v", err)
	}
	if len(*ran) != 2 || !strings.HasPrefix((*ran)[1], "launchctl bootstrap gui/") {
		t.Errorf("ran %v", *ran)
	}

	if code, out, _ := run("daemon", "uninstall"); code != ExitOK || !strings.Contains(out, "recorded activity is kept") {
		t.Errorf("uninstall: code %d, %q", code, out)
	}
}

func TestDaemonInstallRefuses(t *testing.T) {
	// A system that can't tell when you're at the computer.
	daemonTest(t, 0)
	set(t, &idleReader, func() idle.Reader { return func() (time.Duration, error) { return 0, idle.ErrUnsupported } })
	if code, _, errOut := run("daemon", "install"); code != ExitError || !strings.Contains(errOut, "can't tell") {
		t.Errorf("unsupported: code %d, %q", code, errOut)
	}

	// From go run.
	daemonTest(t, 0)
	set(t, &executable, func() (string, error) { return "/tmp/go-build123/exe/txoko", nil })
	if code, _, errOut := run("daemon", "install"); code != ExitError || !strings.Contains(errOut, "built binary") {
		t.Errorf("go run: code %d, %q", code, errOut)
	}

	// launchctl says no.
	daemonTest(t, 0, "launchctl bootstrap")
	if code, _, errOut := run("daemon", "install"); code != ExitError || !strings.Contains(errOut, "starting the service") {
		t.Errorf("launchctl refusing: code %d, %q", code, errOut)
	}

	// No executable, no home, no service on this system.
	daemonTest(t, 0)
	set(t, &executable, func() (string, error) { return "", errors.New("lost") })
	if code, _, _ := run("daemon", "install"); code != ExitError {
		t.Errorf("no executable: code %d", code)
	}
	daemonTest(t, 0)
	set(t, &homeDir, func() (string, error) { return "", errors.New("homeless") })
	for _, cmd := range []string{"install", "uninstall", "status"} {
		if code, _, _ := run("daemon", cmd); code != ExitError {
			t.Errorf("no home, %s: code %d", cmd, code)
		}
	}
	daemonTest(t, 0)
	set(t, &goos, "windows")
	for _, cmd := range []string{"install", "uninstall", "status"} {
		if code, _, errOut := run("daemon", cmd); code != ExitError || !strings.Contains(errOut, "launchd") {
			t.Errorf("windows, %s: code %d, %q", cmd, code, errOut)
		}
	}
}

func TestDaemonStatus(t *testing.T) {
	daemonTest(t, 0)
	if _, out, _ := run("daemon", "status"); !strings.Contains(out, "✓ running — no activity yet today") {
		t.Errorf("running, nothing yet: %q", out)
	}
	st, err := openStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddActive(context.Background(), now().Truncate(time.Hour), 65); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	if _, out, _ := run("daemon", "status"); !strings.Contains(out, "✓ running — 1h05 at the computer today") {
		t.Errorf("running: %q", out)
	}

	daemonTest(t, 0, "launchctl print")
	if _, out, _ := run("daemon", "status"); !strings.Contains(out, "✗ not installed") {
		t.Errorf("not installed: %q", out)
	}
	run("daemon", "install")
	if _, out, _ := run("daemon", "status"); !strings.Contains(out, "✗ installed but not running") || !strings.Contains(out, "txoko.log") {
		t.Errorf("installed, not running: %q", out)
	}
	set(t, &goos, "linux")
	set(t, &serviceRun, func(string, ...string) error { return errors.New("inactive") })
	run("daemon", "install")
	if _, out, _ := run("daemon", "status"); !strings.Contains(out, "journalctl --user -u txoko") {
		t.Errorf("linux, not running: %q", out)
	}
}

// TestDaemonRun runs the real loop for a moment, ticking fast, and checks
// that it recorded active time.
func TestDaemonRun(t *testing.T) {
	daemonTest(t, 5*time.Second)
	set(t, &trackEvery, 5*time.Millisecond)
	set(t, &daemonUntil, func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 100*time.Millisecond)
	})
	if code, _, errOut := run("daemon", "run"); code != ExitOK || errOut != "" {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
	st, err := openStore()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	events, _ := st.Since(context.Background(), time.Time{})
	if len(events) != 1 || events[0].Kind != store.KindActive || events[0].Amount <= 0 {
		t.Errorf("recorded %+v", events)
	}
}

func TestDaemonRunErrors(t *testing.T) {
	daemonTest(t, 0)
	set(t, &idleReader, func() idle.Reader { return func() (time.Duration, error) { return 0, idle.ErrUnsupported } })
	if code, _, errOut := run("daemon", "run"); code != ExitError || !strings.Contains(errOut, "can't tell") {
		t.Errorf("unsupported: code %d, %q", code, errOut)
	}

	// A failing tick is logged, with the time, and the loop goes on.
	daemonTest(t, 0)
	calls := 0
	set(t, &idleReader, func() idle.Reader {
		return func() (time.Duration, error) {
			calls++
			if calls > 1 {
				return 0, errors.New("a hiccup")
			}
			return 0, nil
		}
	})
	set(t, &trackEvery, 5*time.Millisecond)
	set(t, &daemonUntil, func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 50*time.Millisecond)
	})
	if code, _, errOut := run("daemon", "run"); code != ExitOK || !strings.Contains(errOut, "txoko: a hiccup") {
		t.Errorf("hiccup: code %d, %q", code, errOut)
	}

	daemonTest(t, 0)
	t.Setenv("TXOKO_DB", "/dev/null/txoko.db")
	if code, _, _ := run("daemon", "run"); code != ExitError {
		t.Errorf("no database: code %d", code)
	}
}

func TestDaemonUsage(t *testing.T) {
	for _, args := range [][]string{{"daemon"}, {"daemon", "start"}, {"daemon", "install", "now"}} {
		if code, _, _ := run(args...); code != ExitUsage {
			t.Errorf("%v: code %d", args, code)
		}
	}
	if _, out, _ := run("help"); !strings.Contains(out, "daemon install") {
		t.Error("the help lists the daemon")
	}
}

func TestDefaultDaemonHooks(t *testing.T) {
	// The real runner, on a program that does nothing.
	if err := serviceRunDefault("true"); err != nil {
		t.Errorf("running true: %v", err)
	}
	ctx, stop := daemonUntilDefault()
	stop()
	if ctx.Err() == nil {
		t.Error("stopping the context cancels it")
	}
}
