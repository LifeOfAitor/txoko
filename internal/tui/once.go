package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/progress"
	"github.com/LifeOfAitor/txoko/internal/scene"
)

// Default size of the --once scene: small enough for a shell greeting.
const (
	OnceWidth  = 34
	OnceHeight = 14
)

// OnceOptions configures a one-shot, neofetch-style splash.
type OnceOptions struct {
	Scene  string
	Season string // "" follows the calendar
	Seed   uint64 // 0 derives one from the hostname
	Growth float64
	Width  int // scene size; 0 uses OnceWidth and OnceHeight
	Height int
	// Summary adds activity lines (today, streak) when not nil.
	Summary *progress.Summary
	Color   bool
	Now     func() time.Time
}

// Once draws the scene a single time, with a little info panel to its
// right, and writes it to w. It's meant for a shell greeting (.zshrc) or a
// screenshot: no animation, no input, no alternate screen.
func Once(w io.Writer, o OnceOptions) error {
	if o.Width <= 0 || o.Height <= 0 {
		o.Width, o.Height = OnceWidth, OnceHeight
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Seed == 0 {
		o.Seed = DefaultSeed()
	}
	theme := PlainTheme()
	if o.Color {
		theme = DefaultTheme()
	}

	now := o.Now()
	season := scene.SeasonOf(now)
	if s, ok := scene.ParseSeason(o.Season); ok {
		season = s
	}
	night := now.Hour() < 7 || now.Hour() >= 20
	idx, _ := sceneIndex(o.Scene)

	c := canvas.New(o.Width, o.Height)
	scenes[idx].Draw(c, o.Seed, scene.State{
		Growth: clamp01(o.Growth), Night: night, Season: season, MoonPhase: scene.MoonPhaseOf(now),
	})
	art := renderCanvas(c, theme)
	info := onceInfo(theme, scenes[idx].Name(), season, night, clamp01(o.Growth), o.Summary, now)

	// Info sits to the right of the scene, centered vertically.
	top := max(0, (len(art)-len(info))/2)
	var out strings.Builder
	for y, line := range art {
		out.WriteString(line)
		if i := y - top; i >= 0 && i < len(info) {
			out.WriteString("   " + info[i])
		}
		out.WriteString("\n")
	}
	_, err := io.WriteString(w, trimLines(out.String()))
	return err
}

func onceInfo(t Theme, sceneName string, season scene.Season, night bool, growth float64, s *progress.Summary, now time.Time) []string {
	sky := "day"
	if night {
		sky = "night"
	}
	label := func(name string) string { return t.Muted.Render(fmt.Sprintf("%-7s", name)) }
	filled, empty := progressBar(growth, 10)
	moonName, moonEmoji := scene.MoonPhaseName(scene.MoonPhaseOf(now))

	header := t.Title.Render("txoko") + t.Muted.Render(" · "+sceneName+" · "+season.String()+" · "+sky)
	lines := []string{
		header,
		t.Border.Render(strings.Repeat("─", lipgloss.Width(header))),
		label("growth") + t.Ink[canvas.InkLeaf].Render(filled) + t.Muted.Render(empty) + t.Accent.Render(fmt.Sprintf(" %.0f%%", growth*100)),
	}
	if s != nil {
		today := plural(s.Today, "commit")
		if s.FocusToday > 0 {
			today += fmt.Sprintf(" · %d min focus", s.FocusToday)
		}
		lines = append(lines,
			label("today")+t.Accent.Render(today),
			label("streak")+t.Accent.Render(plural(s.Streak, "day")),
		)
	}
	lines = append(lines,
		label("moon")+moonEmoji+" "+t.Accent.Render(moonName),
		label("date")+t.Accent.Render(now.Format("Mon 2 Jan")),
	)
	return lines
}

// trimLines drops trailing spaces from every line, so plain output pastes
// cleanly. Styled output keeps its reset codes at the end.
func trimLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}
