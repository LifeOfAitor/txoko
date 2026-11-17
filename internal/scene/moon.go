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
