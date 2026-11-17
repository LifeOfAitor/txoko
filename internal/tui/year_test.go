package tui

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/LifeOfAitor/txoko/internal/contrib"
)

// sampleCalendar is a year of made-up contributions ending on noon's day,
// read from GitHub an hour before noon.
func sampleCalendar() contrib.Calendar {
	end := contrib.Date(noon())
	c := contrib.Calendar{Source: "@octo on GitHub", Fetched: noon().Add(-time.Hour)}
	for d := end.AddDate(-1, 0, 1); !d.After(end); d = d.AddDate(0, 0, 1) {
		n := 0
		if d.YearDay()%3 == 0 {
			n = d.YearDay() % 7
		}
		c.Days = append(c.Days, contrib.Day{Date: d, Count: n})
	}
	return c
}

func yearOut(t *testing.T, o YearOptions) string {
	t.Helper()
	if o.Now == nil {
		o.Now = noon
	}
	if o.Seed == 0 {
		o.Seed = 42
	}
	if o.MoonPhase == nil {
		o.MoonPhase = &testMoon
	}
	var b bytes.Buffer
	if err := Year(&b, o); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestYearGolden(t *testing.T) {
	assertGolden(t, "year_day", yearOut(t, YearOptions{Calendar: sampleCalendar()}))
	assertGolden(t, "year_night", yearOut(t, YearOptions{Calendar: sampleCalendar(), Sky: "night", Width: 60, Height: 12}))
}

func TestYearInfo(t *testing.T) {
	out := yearOut(t, YearOptions{Calendar: sampleCalendar()})
	cal := sampleCalendar()
	best, _ := cal.BestWeek()
	for _, want := range []string{
		"txoko · your year · @octo on GitHub",
		"total   " + plural(cal.Total(), "contribution") + " in the last year",
		"best    " + strings.TrimSpace(strings.Split(strings.Split(out, "best")[1], "in the")[0]) + " in the week of " + best.Start.Format("2 Jan"),
		"streak  1 day in a row at most",
		"updated 1 hour ago",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	quiet := contrib.Calendar{Source: "nobody", Days: []contrib.Day{{Date: contrib.Date(noon())}}, Fetched: noon()}
	if out := yearOut(t, YearOptions{Calendar: quiet}); strings.Contains(out, "best") {
		t.Errorf("a quiet year has no best week:\n%s", out)
	}
}

func TestYearSizeAndSky(t *testing.T) {
	lines := strings.Split(yearOut(t, YearOptions{Calendar: sampleCalendar()}), "\n")
	if lines[YearHeight] != "" || !strings.Contains(lines[YearHeight+1], "your year") {
		t.Errorf("the numbers go below the %d-row landscape", YearHeight)
	}
	for _, l := range lines {
		if n := len([]rune(l)); n > YearWidth {
			t.Errorf("a line is %d wide, more than %d", n, YearWidth)
		}
	}
	day := yearOut(t, YearOptions{Calendar: sampleCalendar(), Now: midnight, Sky: "day"})
	night := yearOut(t, YearOptions{Calendar: sampleCalendar(), Now: noon, Sky: "night"})
	auto := yearOut(t, YearOptions{Calendar: sampleCalendar(), Now: midnight})
	if strings.Contains(day, "█") || !strings.Contains(night, "█") || !strings.Contains(auto, "█") {
		t.Error("--sky picks day or night; without it, the clock does")
	}
	if strings.Contains(night, "\x1b[") {
		t.Error("plain output has no escape codes")
	}
	if !strings.Contains(yearOut(t, YearOptions{Calendar: sampleCalendar(), Color: true}), "\x1b[") {
		t.Error("colored output has colors")
	}
}

func TestYearDefaults(t *testing.T) {
	var b bytes.Buffer
	if err := Year(&b, YearOptions{Calendar: sampleCalendar()}); err != nil || !strings.Contains(b.String(), "your year") {
		t.Errorf("defaults: %v", err)
	}
	if err := Year(failingWriter{}, YearOptions{Calendar: sampleCalendar()}); err == nil {
		t.Error("a write error is reported")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestAgo(t *testing.T) {
	for d, want := range map[time.Duration]string{
		10 * time.Second: "just now",
		time.Minute:      "1 minute ago",
		45 * time.Minute: "45 minutes ago",
		3 * time.Hour:    "3 hours ago",
		47 * time.Hour:   "47 hours ago",
		72 * time.Hour:   "3 days ago",
	} {
		if got := Ago(d); got != want {
			t.Errorf("Ago(%v) = %q, want %q", d, got, want)
		}
	}
}
