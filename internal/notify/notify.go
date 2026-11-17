// Package notify shows desktop notifications, best effort: if the system
// has no way to show one, nothing happens.
package notify

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// ErrUnsupported means this system has no known way to show notifications.
var ErrUnsupported = errors.New("desktop notifications are not supported here")

// Send shows a notification with a title and a message.
func Send(title, message string) error {
	name, args, err := command(runtime.GOOS, title, message, exec.LookPath)
	if err != nil {
		return err
	}
	return exec.Command(name, args...).Run()
}

// command builds the program and arguments that show a notification on
// goos. It is separate from Send so tests can check it without popping up
// real notifications.
func command(goos, title, message string, lookPath func(string) (string, error)) (string, []string, error) {
	switch goos {
	case "darwin":
		script := "display notification " + appleString(message) +
			" with title " + appleString(title) + ` sound name "Glass"`
		return "osascript", []string{"-e", script}, nil
	case "linux", "freebsd", "openbsd":
		if _, err := lookPath("notify-send"); err != nil {
			return "", nil, ErrUnsupported
		}
		return "notify-send", []string{"--app-name=txoko", title, message}, nil
	}
	return "", nil, ErrUnsupported
}

// appleString quotes s as an AppleScript string literal.
func appleString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
