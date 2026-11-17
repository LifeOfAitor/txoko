package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/contrib"
	"github.com/LifeOfAitor/txoko/internal/scene/year"
)

// Default size of the year landscape when the terminal's isn't known.
const (
	YearWidth  = 80
	YearHeight = 14
)

// YearOptions configures the picture of a year.
type YearOptions struct {
	Calendar contrib.Calendar
	Width    int // 0 uses YearWidth and YearHeight
	Height   int
	Sky      string // "day" or "night"; "" follows the clock
	Seed     uint64 // 0 derives one from the hostname
	Color    bool
	Now      func() time.Time
	// MoonPhase fixes the moon's phase; nil picks a random one.
	MoonPhase *float64
}

// Year draws a year of contributions as a landscape, one column per week,
// with its numbers below, and writes it to w.
func Year(w io.Writer, o YearOptions) error {
	if o.Width <= 0 || o.Height <= 0 {
		o.Width, o.Height = YearWidth, YearHeight
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
	night := now.Hour() < 7 || now.Hour() >= 20
	switch o.Sky {
	case "day":
		night = false
	case "night":
		night = true
	}
	moon := randomMoon()
	if o.MoonPhase != nil {
		moon = *o.MoonPhase
	}

	c := canvas.New(o.Width, o.Height)
	year.Draw(c, o.Seed, o.Calendar.Weeks(), year.State{Night: night, MoonPhase: moon})
	lines := renderCanvas(c, theme)
	lines = append(lines, "")
	lines = append(lines, yearInfo(theme, o.Calendar, now)...)
	_, err := io.WriteString(w, trimLines(strings.Join(lines, "\n")+"\n"))
	return err
}

func yearInfo(t Theme, cal contrib.Calendar, now time.Time) []string {
	label := func(name string) string { return t.Muted.Render(fmt.Sprintf("%-8s", name)) }
	header := t.Title.Render("txoko") + t.Muted.Render(" · your year · "+cal.Source)
	lines := []string{
		header,
		t.Border.Render(strings.Repeat("─", lipgloss.Width(header))),
		label("total") + t.Accent.Render(plural(cal.Total(), "contribution")) + t.Muted.Render(" in the last year"),
	}
	if best, ok := cal.BestWeek(); ok {
		lines = append(lines, label("best")+t.Accent.Render(fmt.Sprint(best.Count))+t.Muted.Render(" in the week of "+best.Start.Format("2 Jan")))
	}
	return append(lines,
		label("streak")+t.Accent.Render(plural(cal.LongestStreak(), "day"))+t.Muted.Render(" in a row at most"),
		label("updated")+t.Muted.Render(Ago(now.Sub(cal.Fetched))),
	)
}

// Ago says how long ago something happened, roughly.
func Ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return plural(int(d/time.Minute), "minute") + " ago"
	case d < 48*time.Hour:
		return plural(int(d/time.Hour), "hour") + " ago"
	}
	return plural(int(d/(24*time.Hour)), "day") + " ago"
}
