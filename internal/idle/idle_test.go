package idle

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// fake replaces the commands idle runs, answering with out and err.
func fake(t *testing.T, out string, err error) *[]string {
	t.Helper()
	var ran []string
	orig := command
	command = func(name string, args ...string) ([]byte, error) {
		ran = append(ran, name+" "+strings.Join(args, " "))
		return []byte(out), err
	}
	t.Cleanup(func() { command = orig })
	return &ran
}

const ioreg = `+-o Root  <class IORegistryEntry>
  +-o IOHIDSystem  <class IOHIDSystem>
      | {
      |   "HIDIdleTime" = 24766369166
      |   "HIDParameters" = {"HIDIdleTime"=0}
      | }
`

func TestMacOS(t *testing.T) {
	ran := fake(t, ioreg, nil)
	d, err := For("darwin")()
	if err != nil || d != 24766369166*time.Nanosecond {
		t.Errorf("got %v, %v", d, err)
	}
	if len(*ran) != 1 || !strings.HasPrefix((*ran)[0], "ioreg -c IOHIDSystem") {
		t.Errorf("ran %v", *ran)
	}

	fake(t, "", errors.New("no ioreg"))
	if _, err := For("darwin")(); err == nil || !strings.Contains(err.Error(), "asking macOS") {
		t.Errorf("a failing ioreg: %v", err)
	}
	fake(t, "nothing useful", nil)
	if _, err := For("darwin")(); err == nil {
		t.Error("an answer without the idle time is an error")
	}
	fake(t, `"HIDIdleTime" = 99999999999999999999999`, nil)
	if _, err := For("darwin")(); err == nil {
		t.Error("a number too big is an error")
	}
}

func TestLinux(t *testing.T) {
	fake(t, "1500\n", nil)
	if d, err := For("linux")(); err != nil || d != 1500*time.Millisecond {
		t.Errorf("got %v, %v", d, err)
	}
	fake(t, "", exec.ErrNotFound)
	if _, err := For("linux")(); !errors.Is(err, ErrUnsupported) {
		t.Errorf("without xprintidle: %v", err)
	}
	fake(t, "", errors.New("no display"))
	if _, err := For("linux")(); err == nil || errors.Is(err, ErrUnsupported) {
		t.Errorf("xprintidle failing: %v", err)
	}
	fake(t, "soon", nil)
	if _, err := For("linux")(); err == nil {
		t.Error("not a number is an error")
	}
}

func TestOtherSystems(t *testing.T) {
	if _, err := For("plan9")(); !errors.Is(err, ErrUnsupported) {
		t.Errorf("got %v", err)
	}
	if System() == nil {
		t.Error("there's always a reader, even if it can't tell")
	}
}

// TestCommand runs a harmless real program through the default runner.
func TestCommand(t *testing.T) {
	if out, err := command("echo", "hi"); err != nil || strings.TrimSpace(string(out)) != "hi" {
		t.Errorf("got %q, %v", out, err)
	}
}
