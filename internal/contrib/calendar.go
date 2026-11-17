// Package contrib reads your contribution calendar, one count per day for
// the last year, and sums it up. It comes from GitHub, or from txoko's own
// record of your activity when you'd rather stay offline.
package contrib

import (
	"time"
)

// Day is one day of the calendar. Date is the calendar day at midnight UTC,
// so that days compare and add up the same in every time zone.
type Day struct {
	Date  time.Time
	Count int
}

// Calendar is a run of consecutive days, oldest first.
type Calendar struct {
	Source  string    // where it comes from, e.g. "@LifeOfAitor on GitHub"
	Days    []Day     // consecutive days, oldest first
	Fetched time.Time // when it was read from its source
}

// Week is seven days of the calendar, starting on Sunday like GitHub's.
type Week struct {
	Start time.Time // the Sunday it starts on
	Count int
}

// Date returns the calendar day of t, in t's own time zone, as a Day date.
func Date(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Total is the sum of every day.
func (c Calendar) Total() int {
	n := 0
	for _, d := range c.Days {
		n += d.Count
	}
	return n
}

// LongestStreak is the most days in a row with something done. A gap in
// the dates breaks a streak too.
func (c Calendar) LongestStreak() int {
	best, run := 0, 0
	for i, d := range c.Days {
		if i > 0 && !d.Date.Equal(c.Days[i-1].Date.AddDate(0, 0, 1)) {
			run = 0
		}
		if d.Count > 0 {
			run++
			best = max(best, run)
		} else {
			run = 0
		}
	}
	return best
}

// Weeks groups the days into weeks starting on Sunday. The first and last
// weeks may be partial.
func (c Calendar) Weeks() []Week {
	var weeks []Week
	for _, d := range c.Days {
		start := d.Date.AddDate(0, 0, -int(d.Date.Weekday()))
		if len(weeks) == 0 || !weeks[len(weeks)-1].Start.Equal(start) {
			weeks = append(weeks, Week{Start: start})
		}
		weeks[len(weeks)-1].Count += d.Count
	}
	return weeks
}

// BestWeek is the week with the most done; ok is false when nothing was.
func (c Calendar) BestWeek() (w Week, ok bool) {
	for _, wk := range c.Weeks() {
		if wk.Count > w.Count {
			w, ok = wk, true
		}
	}
	return w, ok
}

// FromTimes builds a calendar of the days from..to (both included, as
// calendar days in loc) out of the moments something happened, such as
// txoko's own recorded commits and focus sessions.
func FromTimes(times []time.Time, from, to time.Time, loc *time.Location) Calendar {
	first, last := Date(from.In(loc)), Date(to.In(loc))
	index := map[time.Time]int{}
	var c Calendar
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		index[d] = len(c.Days)
		c.Days = append(c.Days, Day{Date: d})
	}
	for _, t := range times {
		if i, ok := index[Date(t.In(loc))]; ok {
			c.Days[i].Count++
		}
	}
	return c
}
