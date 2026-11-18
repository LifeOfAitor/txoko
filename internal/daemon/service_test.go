package daemon

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeService is a service in a throwaway home whose launchctl and
// systemctl calls are only written down, never run.
func fakeService(t *testing.T, goos string, fail map[string]bool) (Service, *[]string) {
	t.Helper()
	var ran []string
	return Service{
		GOOS: goos, Home: t.TempDir(), Exe: "/opt/homebrew/bin/txoko", UID: 501,
		Run: func(name string, args ...string) error {
			cmd := name + " " + strings.Join(args, " ")
			ran = append(ran, cmd)
			for prefix := range fail {
				if strings.HasPrefix(cmd, prefix) {
					return errors.New("refused")
				}
			}
			return nil
		},
	}, &ran
}

func TestLaunchd(t *testing.T) {
	s, ran := fakeService(t, "darwin", nil)
	if err := s.Install(); err != nil {
		t.Fatal(err)
	}
	path, _ := s.File()
	if want := filepath.Join(s.Home, "Library/LaunchAgents", Label+".plist"); path != want {
		t.Errorf("File() = %s", path)
	}
	plist, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := xml.Unmarshal(plist, new(struct{})); err != nil {
		t.Errorf("not a valid plist: %v", err)
	}
	for _, want := range []string{"<string>/opt/homebrew/bin/txoko</string>", "<string>daemon</string>", "<string>run</string>", "RunAtLoad", "KeepAlive", s.LogFile()} {
		if !strings.Contains(string(plist), want) {
			t.Errorf("plist lacks %q", want)
		}
	}
	if want := []string{
		"launchctl bootout gui/501/" + Label,
		"launchctl bootstrap gui/501 " + path,
	}; strings.Join(*ran, "\n") != strings.Join(want, "\n") {
		t.Errorf("ran:\n%s", strings.Join(*ran, "\n"))
	}

	st, err := s.Check()
	if err != nil || !st.Installed || !st.Running {
		t.Errorf("Check() = %+v, %v", st, err)
	}

	*ran = nil
	if err := s.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Error("the plist should be gone")
	}
	if len(*ran) != 1 || !strings.HasPrefix((*ran)[0], "launchctl bootout") {
		t.Errorf("ran %v", *ran)
	}
	if err := s.Uninstall(); err != nil {
		t.Errorf("uninstalling twice is fine: %v", err)
	}
}

func TestSystemd(t *testing.T) {
	s, ran := fakeService(t, "linux", nil)
	if err := s.Install(); err != nil {
		t.Fatal(err)
	}
	path, _ := s.File()
	unit, _ := os.ReadFile(path)
	if !strings.Contains(string(unit), `ExecStart="/opt/homebrew/bin/txoko" daemon run`) || !strings.Contains(string(unit), "WantedBy=default.target") {
		t.Errorf("unit:\n%s", unit)
	}
	if want := "systemctl --user daemon-reload\nsystemctl --user enable --now txoko.service"; strings.Join(*ran, "\n") != want {
		t.Errorf("ran:\n%s", strings.Join(*ran, "\n"))
	}
	*ran = nil
	if err := s.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if want := "systemctl --user disable --now txoko.service\nsystemctl --user daemon-reload"; strings.Join(*ran, "\n") != want {
		t.Errorf("ran:\n%s", strings.Join(*ran, "\n"))
	}
	st, _ := s.Check()
	if st.Installed || !st.Running { // the fake systemctl says it's active
		t.Errorf("Check() = %+v", st)
	}
}

func TestServiceFailures(t *testing.T) {
	s, _ := fakeService(t, "darwin", map[string]bool{"launchctl bootstrap": true, "launchctl print": true})
	if err := s.Install(); err == nil || !strings.Contains(err.Error(), "starting the service") {
		t.Errorf("launchctl refusing: %v", err)
	}
	if st, _ := s.Check(); !st.Installed || st.Running {
		t.Errorf("installed but not running: %+v", st)
	}

	for _, prefix := range []string{"systemctl --user daemon-reload", "systemctl --user enable"} {
		s, _ := fakeService(t, "linux", map[string]bool{prefix: true})
		if err := s.Install(); err == nil {
			t.Errorf("%s refusing should fail the install", prefix)
		}
	}

	// A home where nothing can be written.
	s, _ = fakeService(t, "darwin", nil)
	s.Home = "/dev/null"
	if err := s.Install(); err == nil {
		t.Error("an unwritable home is an error")
	}
	s, _ = fakeService(t, "darwin", nil)
	dir := filepath.Join(s.Home, "Library", "LaunchAgents")
	if err := os.MkdirAll(filepath.Join(dir, Label+".plist"), 0o755); err != nil { // a folder in the way
		t.Fatal(err)
	}
	if err := s.Install(); err == nil || !strings.Contains(err.Error(), "writing the service") {
		t.Errorf("a folder where the plist goes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, Label+".plist", "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Uninstall(); err == nil || !strings.Contains(err.Error(), "removing the service") {
		t.Errorf("a folder that can't be removed: %v", err)
	}
}

func TestNoService(t *testing.T) {
	s, _ := fakeService(t, "windows", nil)
	for name, err := range map[string]error{
		"Install":   s.Install(),
		"Uninstall": s.Uninstall(),
	} {
		if !errors.Is(err, ErrNoService) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := s.Check(); !errors.Is(err, ErrNoService) {
		t.Errorf("Check: %v", err)
	}
}

func TestPlistEscapesPaths(t *testing.T) {
	s, _ := fakeService(t, "darwin", nil)
	s.Exe = "/Users/me/Apps & Tools/txoko"
	if !strings.Contains(s.definition(), "Apps &amp; Tools") {
		t.Error("an & in the path must be escaped in the plist")
	}
}
