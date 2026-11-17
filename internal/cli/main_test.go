package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
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

	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}
