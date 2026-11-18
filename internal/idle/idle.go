// Package idle tells how long it's been since you last touched the
// keyboard or the mouse. It asks the system, the same way screen savers do,
// and learns nothing else: not which app you use or what you type.
package idle

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ErrUnsupported means this system has no way txoko knows to tell.
var ErrUnsupported = errors.New("can't tell when you're at the computer here: txoko knows macOS, and Linux with xprintidle (X11)")

// Reader returns the time since the last input.
type Reader func() (time.Duration, error)

// command runs a program and returns what it printed. Tests replace it.
var command = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// System returns the reader for the system txoko runs on.
func System() Reader { return For(runtime.GOOS) }

// For returns the reader for a given GOOS.
func For(goos string) Reader {
	switch goos {
	case "darwin":
		// IOKit's HID system keeps the time since the last input, in ns.
		return func() (time.Duration, error) {
			out, err := command("ioreg", "-c", "IOHIDSystem", "-d", "4")
			if err != nil {
				return 0, fmt.Errorf("asking macOS for the idle time: %w", err)
			}
			return parseIOReg(out)
		}
	case "linux":
		// xprintidle prints the X11 idle time, in ms.
		return func() (time.Duration, error) {
			out, err := command("xprintidle")
			if errors.Is(err, exec.ErrNotFound) {
				return 0, ErrUnsupported
			}
			if err != nil {
				return 0, fmt.Errorf("running xprintidle: %w", err)
			}
			return parseMillis(out)
		}
	}
	return func() (time.Duration, error) { return 0, ErrUnsupported }
}

var hidIdle = regexp.MustCompile(`"HIDIdleTime" = (\d+)`)

// parseIOReg reads HIDIdleTime out of ioreg's output.
func parseIOReg(out []byte) (time.Duration, error) {
	m := hidIdle.FindSubmatch(out)
	if m == nil {
		return 0, errors.New("macOS didn't say how long it's been idle")
	}
	ns, err := strconv.ParseInt(string(m[1]), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("reading the idle time: %w", err)
	}
	return time.Duration(ns), nil
}

// parseMillis reads a number of milliseconds.
func parseMillis(out []byte) (time.Duration, error) {
	ms, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("reading the idle time: %w", err)
	}
	return time.Duration(ms) * time.Millisecond, nil
}
