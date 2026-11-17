package notify

import (
	"errors"
	"strings"
	"testing"
)

func found(string) (string, error)   { return "/usr/bin/notify-send", nil }
func missing(string) (string, error) { return "", errors.New("not found") }

func TestCommandDarwin(t *testing.T) {
	name, args, err := command("darwin", "txoko", `Say "hi" \o/`, missing)
	if err != nil || name != "osascript" || len(args) != 2 || args[0] != "-e" {
		t.Fatalf("command = %s %v, %v", name, args, err)
	}
	want := `display notification "Say \"hi\" \\o/" with title "txoko" sound name "Glass"`
	if args[1] != want {
		t.Errorf("script =\n%s\nwant\n%s", args[1], want)
	}
}

func TestCommandLinux(t *testing.T) {
	name, args, err := command("linux", "txoko", "done", found)
	if err != nil || name != "notify-send" || strings.Join(args, "|") != "--app-name=txoko|txoko|done" {
		t.Errorf("command = %s %v, %v", name, args, err)
	}
	if _, _, err := command("linux", "t", "m", missing); !errors.Is(err, ErrUnsupported) {
		t.Errorf("without notify-send: %v", err)
	}
}

func TestCommandUnsupported(t *testing.T) {
	if _, _, err := command("plan9", "t", "m", found); !errors.Is(err, ErrUnsupported) {
		t.Errorf("plan9: %v", err)
	}
}
