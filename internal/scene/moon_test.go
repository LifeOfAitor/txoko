package scene

import (
	"math"
	"testing"
	"time"
)

func TestMoonPhaseOf(t *testing.T) {
	// Real moon phases (UTC) from published tables.
	tests := []struct {
		name string
		at   time.Time
		want float64
	}{
		{"reference new moon", time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC), 0},
		{"full moon 2025-01-13", time.Date(2025, 1, 13, 22, 27, 0, 0, time.UTC), 0.5},
		{"new moon 2025-03-29", time.Date(2025, 3, 29, 10, 58, 0, 0, time.UTC), 0},
		{"first quarter 2025-06-03", time.Date(2025, 6, 3, 3, 41, 0, 0, time.UTC), 0.25},
		{"last quarter 2025-09-14", time.Date(2025, 9, 14, 10, 33, 0, 0, time.UTC), 0.75},
	}
	for _, tt := range tests {
		got := MoonPhaseOf(tt.at)
		// Distance on the circle, so 0.99 counts as close to 0.
		d := math.Abs(got - tt.want)
		d = math.Min(d, 1-d)
		if d > 0.04 { // about a day
			t.Errorf("%s: phase %.3f, want %.2f", tt.name, got, tt.want)
		}
	}
}

func TestMoonPhaseRange(t *testing.T) {
	start := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	for h := 0; h < 24*365*40; h += 97 {
		if p := MoonPhaseOf(start.Add(time.Duration(h) * time.Hour)); p < 0 || p >= 1 {
			t.Fatalf("phase out of range: %v", p)
		}
	}
}

func TestMoonPhaseName(t *testing.T) {
	for _, tt := range []struct {
		phase       float64
		name, emoji string
	}{
		{0, "new moon", "🌑"},
		{0.97, "new moon", "🌑"},
		{0.12, "waxing crescent", "🌒"},
		{0.25, "first quarter", "🌓"},
		{0.38, "waxing gibbous", "🌔"},
		{0.5, "full moon", "🌕"},
		{0.62, "waning gibbous", "🌖"},
		{0.75, "last quarter", "🌗"},
		{0.87, "waning crescent", "🌘"},
		{-0.01, "new moon", "🌑"},
	} {
		name, emoji := MoonPhaseName(tt.phase)
		if name != tt.name || emoji != tt.emoji {
			t.Errorf("MoonPhaseName(%v) = %s %s, want %s %s", tt.phase, name, emoji, tt.name, tt.emoji)
		}
	}
}
