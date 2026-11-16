package tui

import (
	"errors"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
)

// ErrNotTerminal is returned when stdout is not an interactive terminal,
// for example when txoko's output is piped into another command.
var ErrNotTerminal = errors.New("the live view needs an interactive terminal")

// Run starts the interactive UI and blocks until the user quits.
func Run(opts Options) error {
	if !term.IsTerminal(os.Stdout.Fd()) {
		return ErrNotTerminal
	}
	_, err := tea.NewProgram(New(opts)).Run()
	return err
}
