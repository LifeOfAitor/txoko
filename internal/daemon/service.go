package daemon

import (
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strconv"
)

// Label names txoko's service for launchd and systemd.
const Label = "io.github.lifeofaitor.txoko"

// ErrNoService means txoko can't run as a service on this system.
var ErrNoService = errors.New("txoko can run in the background on macOS (launchd) and Linux (systemd)")

// Service installs, checks and removes the background service, which runs
// `txoko daemon run` from the moment you log in.
type Service struct {
	GOOS string // "darwin" or "linux"
	Home string // your home folder
	Exe  string // the txoko to run
	UID  int    // your user id, for launchd
	// Run runs launchctl or systemctl. Tests replace it.
	Run func(name string, args ...string) error
}

// File is where the service's definition lives.
func (s Service) File() (string, error) {
	switch s.GOOS {
	case "darwin":
		return filepath.Join(s.Home, "Library", "LaunchAgents", Label+".plist"), nil
	case "linux":
		return filepath.Join(s.Home, ".config", "systemd", "user", "txoko.service"), nil
	}
	return "", ErrNoService
}

// LogFile is where the service writes its errors, on macOS. On Linux
// they go to the journal: journalctl --user -u txoko.
func (s Service) LogFile() string {
	return filepath.Join(s.Home, "Library", "Logs", "txoko.log")
}

// Install writes the service's definition and starts it. Installing again
// replaces it, for instance to point it at a new txoko.
func (s Service) Install() error {
	path, err := s.File()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(s.definition()), 0o644); err != nil {
		return fmt.Errorf("writing the service: %w", err)
	}
	if s.GOOS == "darwin" {
		_ = s.Run("launchctl", "bootout", s.domain()+"/"+Label) // an older one may be running
		if err := s.Run("launchctl", "bootstrap", s.domain(), path); err != nil {
			return fmt.Errorf("starting the service: %w", err)
		}
		return nil
	}
	if err := s.Run("systemctl", "--user", "daemon-reload"); err != nil {
		return fmt.Errorf("starting the service: %w", err)
	}
	if err := s.Run("systemctl", "--user", "enable", "--now", "txoko.service"); err != nil {
		return fmt.Errorf("starting the service: %w", err)
	}
	return nil
}

// Uninstall stops the service and removes its definition. Your recorded
// activity is kept.
func (s Service) Uninstall() error {
	path, err := s.File()
	if err != nil {
		return err
	}
	if s.GOOS == "darwin" {
		_ = s.Run("launchctl", "bootout", s.domain()+"/"+Label)
	} else {
		_ = s.Run("systemctl", "--user", "disable", "--now", "txoko.service")
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing the service: %w", err)
	}
	if s.GOOS == "linux" {
		_ = s.Run("systemctl", "--user", "daemon-reload")
	}
	return nil
}

// Status is whether the service is installed and running.
type Status struct {
	Installed bool
	Running   bool
}

// Check reports the service's status.
func (s Service) Check() (Status, error) {
	path, err := s.File()
	if err != nil {
		return Status{}, err
	}
	var st Status
	if _, err := os.Stat(path); err == nil {
		st.Installed = true
	}
	if s.GOOS == "darwin" {
		st.Running = s.Run("launchctl", "print", s.domain()+"/"+Label) == nil
	} else {
		st.Running = s.Run("systemctl", "--user", "is-active", "--quiet", "txoko.service") == nil
	}
	return st, nil
}

func (s Service) domain() string { return "gui/" + strconv.Itoa(s.UID) }

// definition is the launchd property list or the systemd unit.
func (s Service) definition() string {
	if s.GOOS == "darwin" {
		return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<!-- txoko: records the minutes you're at the computer. Remove it with: txoko daemon uninstall -->
	<key>Label</key>
	<string>` + Label + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + html.EscapeString(s.Exe) + `</string>
		<string>daemon</string>
		<string>run</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>ProcessType</key>
	<string>Background</string>
	<key>StandardErrorPath</key>
	<string>` + html.EscapeString(s.LogFile()) + `</string>
</dict>
</plist>
`
	}
	return `# txoko: records the minutes you're at the computer.
# Remove it with: txoko daemon uninstall
[Unit]
Description=txoko active time

[Service]
ExecStart="` + s.Exe + `" daemon run
Restart=on-failure
RestartSec=30

[Install]
WantedBy=default.target
`
}
