// Package progress turns raw activity into the numbers a scene shows:
// how grown it is, today's activity and your streak.
//
// The growth model is meant to feel alive rather than to keep score:
//
//   - every commit adds one point, every 25 minutes of completed pomodoro
//     focus add one more, and so does every hour at the computer;
//   - points fade with a half-life of two weeks, so the tree reflects your
//     recent rhythm — keep going and it stays lush, take a break and it
//     slowly thins out (it never dies);
//   - growth = 1 − e^(−points/Scale): early commits make a big difference
//     and it approaches full bloom gently instead of hitting a hard cap.
//
// With Scale = 25, about three commits a day keep the tree near 90%.
package progress

import (
	"fmt"
	"math"
	"time"

	"github.com/LifeOfAitor/txoko/internal/store"
)

// Tuning knobs for the growth curve.
const (
	HalfLife = 14 * 24 * time.Hour // how long until a commit counts half
	Scale    = 25.0                // points for ~63% growth
	Window   = 120 * 24 * time.Hour
	// Window is how far back events matter: older ones weigh under 0.3%.

	// FocusMinutesPerPoint makes a classic 25-minute pomodoro worth one
	// commit.
	FocusMinutesPerPoint = 25.0

	// ActiveMinutesPerPoint makes an hour at the computer worth one commit:
	// a working day keeps the tree healthy, without outweighing commits.
	ActiveMinutesPerPoint = 60.0
)

// Summary is what the UI shows.
type Summary struct {
	Growth      float64 // 0..1
	Today       int     // commits since local midnight
	FocusToday  int     // minutes of completed focus since local midnight
	ActiveToday int     // minutes at the computer since local midnight
	Streak      int     // consecutive days with any activity
	Recent      int     // events inside Window
}

// points is how much an event feeds growth when fresh.
func points(e store.Event) float64 {
	amount := e.Amount
	if amount == 0 {
		amount = 1
	}
	switch e.Kind {
	case store.KindFocus:
		return amount / FocusMinutesPerPoint
	case store.KindActive:
		return amount / ActiveMinutesPerPoint
	}
	return amount
}

// Compute summarizes events as of now. Days follow now's time zone.
func Compute(events []store.Event, now time.Time) Summary {
	var (
		s      Summary
		total  float64
		days   = map[civilDay]bool{}
		today  = dayOf(now)
		cutoff = now.Add(-Window)
	)
	for _, e := range events {
		if e.At.Before(cutoff) || e.At.After(now) {
			continue
		}
		s.Recent++
		age := now.Sub(e.At)
		total += points(e) * math.Exp2(-float64(age)/float64(HalfLife))

		d := dayOf(e.At.In(now.Location()))
		days[d] = true
		if d != today {
			continue
		}
		switch e.Kind {
		case store.KindFocus:
			s.FocusToday += int(math.Round(e.Amount))
		case store.KindActive:
			s.ActiveToday += int(math.Round(e.Amount))
		default:
			s.Today++
		}
	}

	s.Growth = 1 - math.Exp(-total/Scale)

	// A streak is still alive today even before today's first commit.
	day := today
	if !days[day] {
		day = day.prev()
	}
	for days[day] {
		s.Streak++
		day = day.prev()
	}
	return s
}

// civilDay is a calendar date, independent of the time of day.
type civilDay struct {
	y int
	m time.Month
	d int
}

func dayOf(t time.Time) civilDay {
	y, m, d := t.Date()
	return civilDay{y, m, d}
}

func (c civilDay) prev() civilDay {
	// Noon avoids daylight-saving edge cases when stepping back a day.
	return dayOf(time.Date(c.y, c.m, c.d, 12, 0, 0, 0, time.UTC).AddDate(0, 0, -1))
}

// Clock writes minutes the short way: "45m", "2h", "2h10".
func Clock(minutes int) string {
	switch h, m := minutes/60, minutes%60; {
	case h == 0:
		return fmt.Sprintf("%dm", m)
	case m == 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dh%02d", h, m)
	}
}
