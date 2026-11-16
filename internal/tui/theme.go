package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/LifeOfAitor/txoko/internal/canvas"
)

// Theme maps semantic inks and UI parts to terminal styles.
type Theme struct {
	Ink    [canvas.InkCount]lipgloss.Style
	Border lipgloss.Style
	Title  lipgloss.Style
	Muted  lipgloss.Style
	Accent lipgloss.Style
}

// DefaultTheme uses only the 16 basic ANSI colors. Those are the colors your
// terminal theme defines, so txoko automatically matches Catppuccin, Gruvbox,
// Rosé Pine or whatever palette you run.
func DefaultTheme() Theme {
	fg := func(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

	var t Theme
	t.Ink[canvas.InkSoil] = fg(lipgloss.Yellow).Faint(true)
	t.Ink[canvas.InkGrass] = fg(lipgloss.Green)
	t.Ink[canvas.InkStem] = fg(lipgloss.Green)
	t.Ink[canvas.InkLeaf] = fg(lipgloss.BrightGreen)
	t.Ink[canvas.InkBud] = fg(lipgloss.BrightGreen)
	t.Ink[canvas.InkPetalPink] = fg(lipgloss.BrightMagenta)
	t.Ink[canvas.InkPetalRed] = fg(lipgloss.BrightRed)
	t.Ink[canvas.InkPetalYellow] = fg(lipgloss.Yellow)
	t.Ink[canvas.InkCenter] = fg(lipgloss.BrightYellow)
	t.Ink[canvas.InkStar] = fg(lipgloss.White)
	t.Ink[canvas.InkMoon] = fg(lipgloss.BrightYellow)
	t.Ink[canvas.InkSun] = fg(lipgloss.BrightYellow)
	t.Ink[canvas.InkCloud] = fg(lipgloss.White).Faint(true)
	t.Ink[canvas.InkFirefly] = fg(lipgloss.BrightYellow)
	t.Ink[canvas.InkButterfly] = fg(lipgloss.Magenta)
	t.Ink[canvas.InkWood] = fg(lipgloss.Yellow)
	t.Ink[canvas.InkWoodDim] = fg(lipgloss.Yellow).Faint(true)
	t.Ink[canvas.InkLeafDark] = fg(lipgloss.Green)
	t.Ink[canvas.InkLeafDim] = fg(lipgloss.Green).Faint(true)
	t.Ink[canvas.InkPot] = fg(lipgloss.White)

	t.Border = fg(lipgloss.BrightBlack)
	t.Title = fg(lipgloss.Green).Bold(true)
	t.Muted = fg(lipgloss.BrightBlack)
	t.Accent = fg(lipgloss.Yellow)
	return t
}

// PlainTheme has no styling at all. Tests use it to compare plain text.
func PlainTheme() Theme { return Theme{} }

// renderCanvas turns a canvas into styled lines. Consecutive cells sharing
// an ink are styled as one run, which keeps the output small: one escape
// sequence per run instead of one per character.
func renderCanvas(c *canvas.Canvas, t Theme) []string {
	lines := make([]string, c.Height())
	var line, run strings.Builder
	for y := range c.Height() {
		line.Reset()
		row := c.Row(y)
		for i := 0; i < len(row); {
			ink := row[i].Ink
			run.Reset()
			for ; i < len(row) && row[i].Ink == ink; i++ {
				run.WriteRune(row[i].Rune)
			}
			if ink == canvas.InkNone {
				line.WriteString(run.String())
			} else {
				line.WriteString(t.Ink[ink].Render(run.String()))
			}
		}
		lines[y] = line.String()
	}
	return lines
}
