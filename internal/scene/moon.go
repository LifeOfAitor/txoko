package scene

import (
	"math"
	"time"
)

// synodicMonth is the average time between two new moons.
const synodicMonth = time.Duration(29.530588853 * 24 * float64(time.Hour))

// knownNewMoon is a reference new moon: 6 January 2000, 18:14 UTC.
var knownNewMoon = time.Date(2000, time.January, 6, 18, 14, 0, 0, time.UTC)

// MoonPhaseOf returns how far the moon is through its cycle at t, from 0
// (new moon) through 0.25 (first quarter), 0.5 (full) and 0.75 (last
// quarter) back to 1. It's accurate to within about a day, plenty for
// drawing.
func MoonPhaseOf(t time.Time) float64 {
	cycles := float64(t.Sub(knownNewMoon)) / float64(synodicMonth)
	return cycles - math.Floor(cycles)
}

// moonPhases names the eight traditional phases, with their emoji.
var moonPhases = []struct{ name, emoji string }{
	{"new moon", "🌑"},
	{"waxing crescent", "🌒"},
	{"first quarter", "🌓"},
	{"waxing gibbous", "🌔"},
	{"full moon", "🌕"},
	{"waning gibbous", "🌖"},
	{"last quarter", "🌗"},
	{"waning crescent", "🌘"},
}

// MoonPhaseName returns the traditional name and emoji of a phase from
// MoonPhaseOf, e.g. "first quarter" and "🌓". Each name covers an eighth of
// the cycle, centered on its exact moment.
func MoonPhaseName(phase float64) (name, emoji string) {
	i := int(math.Floor(phase*8+0.5)) & 7 // wraps 8 back to 0
	return moonPhases[i].name, moonPhases[i].emoji
}
