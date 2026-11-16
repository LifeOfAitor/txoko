package tui

import (
	"errors"
	"os"
	"testing"

	"github.com/charmbracelet/x/term"
)

func TestRunRequiresTerminal(t *testing.T) {
	if term.IsTerminal(os.Stdout.Fd()) {
		t.Skip("stdout is a terminal; this test checks the non-interactive path")
	}
	if err := Run(Options{}); !errors.Is(err, ErrNotTerminal) {
		t.Errorf("Run() error = %v, want ErrNotTerminal", err)
	}
}
