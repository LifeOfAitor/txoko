package year

import (
	"strings"
	"testing"
	"time"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/contrib"
	"github.com/LifeOfAitor/txoko/internal/testutil/golden"
)

const testSeed = 3

// sampleYear is 53 weeks from Sunday 17 Nov 2024, with quiet stretches,
// busy ones and one standout week.
func sampleYear() []contrib.Week {
	counts := []int{
		0, 2, 5, 9, 0, 0, 3, 12, 20, 4, 1, 0, 0, 7, // winter
		15, 30, 8, 2, 0, 0, 0, 6, 11, 25, 40, 18, 3, // spring and early summer
		0, 1, 0, 0, 2, 9, 14, 6, 0, 0, 5, 22, 60, // summer
		33, 12, 7, 0, 0, 4, 10, 16, 8, 2, 0, 1, 19, // autumn
	}
	start := time.Date(2024, 11, 17, 0, 0, 0, 0, time.UTC)
	weeks := make([]contrib.Week, len(counts))
	for i, n := range counts {
		weeks[i] = contrib.Week{Start: start.AddDate(0, 0, 7*i), Count: n}
	}
	return weeks
}

func render(w, h int, weeks []contrib.Week, s State) *canvas.Canvas {
	c := canvas.New(w, h)
	Draw(c, testSeed, weeks, s)
	return c
}

func TestGolden(t *testing.T) {
	for _, tt := range []struct {
		name string
		w, h int
		s    State
	}{
		{"day", 110, 14, State{Frame: 30}},
		{"night", 110, 14, State{Night: true, MoonPhase: 0.4}},
		{"narrow", 60, 12, State{}},
		{"tall", 80, 22, State{Night: true, MoonPhase: 0.7}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			golden.Assert(t, tt.name, render(tt.w, tt.h, sampleYear(), tt.s).String())
		})
	}
}

// TestBusierGrowsTaller: the busiest week grows the tallest, and a week
// with nothing done is bare earth.
func TestBusierGrowsTaller(t *testing.T) {
	weeks := sampleYear()
	l := newLayout(110, 14, weeks)
	busiest, quietest := 0, 0
	for i, wk := range weeks {
		if wk.Count > weeks[busiest].Count {
			busiest = i
		}
		if wk.Count == 0 && l.heights[i] != 0 {
			t.Errorf("week %d did nothing but grew %d rows", i, l.heights[i])
		}
		if wk.Count > 0 && l.heights[i] == 0 {
			t.Errorf("week %d did %d but grew nothing", i, wk.Count)
		}
		if wk.Count < weeks[quietest].Count {
			quietest = i
		}
	}
	for i := range weeks {
		if l.heights[i] > l.heights[busiest] {
			t.Errorf("week %d grew taller than the busiest", i)
		}
		for j := range weeks {
			if weeks[i].Count > weeks[j].Count && l.heights[i] < l.heights[j] {
				t.Errorf("week %d did more than week %d but grew less", i, j)
			}
		}
	}
	if l.heights[busiest] <= bushRows {
		t.Error("the busiest week should grow a tree")
	}
	if l.sky[l.x(busiest)] < 2 {
		t.Error("even the busiest week leaves some sky")
	}
}

func inksOfWeek(c *canvas.Canvas, l layout, i int) map[canvas.Ink]int {
	n := map[canvas.Ink]int{}
	for y := 0; y < l.ground; y++ {
		for x := l.x(i); x < l.x(i)+l.cw; x++ {
			n[c.At(x, y).Ink]++
		}
	}
	return n
}

// TestEachWeekInItsSeason: a winter week is snowy, a spring one blossoms,
// an autumn one turns red and gold.
func TestEachWeekInItsSeason(t *testing.T) {
	weeks := sampleYear()
	c := render(110, 14, weeks, State{})
	l := newLayout(110, 14, weeks)
	at := func(month time.Month) int { // the busiest week of that month
		best := -1
		for i, wk := range weeks {
			if wk.Start.Month() == month && (best < 0 || wk.Count > weeks[best].Count) {
				best = i
			}
		}
		return best
	}
	if n := inksOfWeek(c, l, at(time.January)); n[canvas.InkSnow] == 0 || n[canvas.InkLeaf] != 0 {
		t.Errorf("January: %v", n)
	}
	if n := inksOfWeek(c, l, at(time.May)); n[canvas.InkBlossom]+n[canvas.InkBlossomLight] == 0 {
		t.Errorf("May: %v", n)
	}
	if n := inksOfWeek(c, l, at(time.July)); n[canvas.InkLeaf]+n[canvas.InkLeafDark] == 0 {
		t.Errorf("July: %v", n)
	}
	if n := inksOfWeek(c, l, at(time.October)); n[canvas.InkAutumnRed]+n[canvas.InkAutumnOrange]+n[canvas.InkAutumnGold] == 0 {
		t.Errorf("October: %v", n)
	}
}

// TestMonths: the month names run along the bottom, each under the week the
// month starts in, without running into each other.
func TestMonths(t *testing.T) {
	weeks := sampleYear()
	c := render(110, 14, weeks, State{})
	l := newLayout(110, 14, weeks)
	row := strings.TrimRight(func() string {
		var b strings.Builder
		for _, cell := range c.Row(13) {
			b.WriteRune(cell.Rune)
		}
		return b.String()
	}(), " ")
	for _, m := range []string{"Nov", "Dec", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct"} {
		if !strings.Contains(row, m) {
			t.Errorf("no %s in %q", m, row)
		}
	}
	// 1 Jan 2025 falls in the week of Sunday 29 Dec.
	for i, wk := range weeks {
		if wk.Start.Equal(time.Date(2024, 12, 29, 0, 0, 0, 0, time.UTC)) && !strings.HasPrefix(row[l.x(i):], "Jan") {
			t.Errorf("Jan should start at column %d: %q", l.x(i), row)
		}
	}
	// On a narrow canvas some names give way, but none overlap.
	narrow := render(40, 12, weeks, State{})
	var b strings.Builder
	for _, cell := range narrow.Row(11) {
		b.WriteRune(cell.Rune)
	}
	for _, word := range strings.Fields(b.String()) {
		if len(word) != 3 {
			t.Errorf("month names ran into each other: %q", b.String())
		}
	}
}

// TestSky: the night sky stays off the plants, and the moon hangs over the
// quietest stretch rather than over the busiest week.
func TestSky(t *testing.T) {
	weeks := sampleYear()
	c := render(110, 14, weeks, State{Night: true, MoonPhase: 0.5})
	l := newLayout(110, 14, weeks)
	moonX := -1
	for y := range c.Height() {
		for x, cell := range c.Row(y) {
			switch cell.Ink {
			case canvas.InkStar, canvas.InkStarDim, canvas.InkStarBright:
				if l.UnderCanopy(x, y, 0) {
					t.Errorf("a star at (%d,%d) is on a plant", x, y)
				}
			case canvas.InkMoon:
				moonX = x
				if l.UnderCanopy(x, y, 0) {
					t.Errorf("the moon at (%d,%d) is on a plant", x, y)
				}
			}
		}
	}
	busiest := l.x(38) // 60, the standout week
	if moonX < 0 || moonX >= busiest-2 && moonX <= busiest+l.cw+2 {
		t.Errorf("the moon (column %d) should keep away from the busiest week (column %d)", moonX, busiest)
	}
	// With no clear sky anywhere, it goes back to its corner.
	full := make([]contrib.Week, 53)
	for i := range full {
		full[i] = contrib.Week{Start: weeks[0].Start.AddDate(0, 0, 7*i), Count: 10}
	}
	if _, _, r := newLayout(60, 8, full).MoonFrame(); r != 0 {
		t.Errorf("no room for a framed moon, got radius %d", r)
	}
}

func TestNothingToShow(t *testing.T) {
	if s := render(80, 14, nil, State{}).String(); strings.TrimSpace(s) != "" {
		t.Error("no weeks, nothing drawn")
	}
	quiet := sampleYear()
	for i := range quiet {
		quiet[i].Count = 0
	}
	c := render(80, 14, quiet, State{})
	l := newLayout(80, 14, quiet)
	for y := 0; y < l.ground; y++ {
		for _, cell := range c.Row(y) {
			if cell.Ink != canvas.InkNone && cell.Ink != canvas.InkCloud {
				t.Fatalf("a quiet year is bare earth, found %q", cell.Rune)
			}
		}
	}
}

func TestAnySizeIsSafe(_ *testing.T) {
	weeks := sampleYear()
	for w := 0; w < 200; w += 7 {
		for h := 0; h < 30; h += 2 {
			for _, n := range []int{1, 5, 53} {
				Draw(canvas.New(w, h), uint64(w+h), weeks[:n], State{Night: h%4 == 0, MoonPhase: .3})
			}
		}
	}
}

func TestDeterministic(t *testing.T) {
	s := State{Night: true, MoonPhase: 0.3, Frame: 9}
	a, b := render(90, 14, sampleYear(), s).String(), render(90, 14, sampleYear(), s).String()
	if a != b {
		t.Error("same inputs drew different years")
	}
}
