package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LifeOfAitor/txoko/internal/idle"
)

// TestMain points every path txoko uses — database, config and git's global
// config — into a throwaway folder, so tests never touch the developer's
// real data or ~/.gitconfig.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "txoko-cli-test")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	gitconfig := filepath.Join(tmp, "gitconfig")
	_ = os.WriteFile(gitconfig, nil, 0o644)
	for k, v := range map[string]string{
		"TXOKO_DB":            filepath.Join(tmp, "data", "txoko.db"),
		"XDG_CONFIG_HOME":     filepath.Join(tmp, "config"),
		"GIT_CONFIG_GLOBAL":   gitconfig,
		"GIT_CONFIG_NOSYSTEM": "1",
	} {
		_ = os.Setenv(k, v)
	}
	// Never pop up real desktop notifications from tests.
	notifySend = func(string, string) error { return nil }
	// Nor depend on whether a txoko happens to be on the developer's PATH.
	lookPath = func(string) (string, error) { return "", os.ErrNotExist }
	// Nor ever install a real background service, or read the real idle time.
	serviceRun = func(string, ...string) error { return errors.New("no services in tests") }
	homeDir = func() (string, error) { return filepath.Join(tmp, "home"), nil }
	idleReader = func() idle.Reader { return func() (time.Duration, error) { return 0, idle.ErrUnsupported } }

	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}
