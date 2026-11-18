package progress

import (
	"math"
	"testing"
	"time"

	"github.com/LifeOfAitor/txoko/internal/store"
)

var (
	madrid, _ = time.LoadLocation("Europe/Madrid")
	now       = time.Date(2026, 11, 16, 21, 0, 0, 0, madrid)
)

func commitsAt(times ...time.Time) []store.Event {
	events := make([]store.Event, len(times))
	for i, t := range times {
		events[i] = store.Event{Kind: store.KindCommit, Repo: "r", Ref: t.String(), At: t}
	}
	return events
}

func daysAgo(d int, hour int) time.Time {
	y, m, day := now.AddDate(0, 0, -d).Date()
	return time.Date(y, m, day, hour, 0, 0, 0, madrid)
}

func TestEmpty(t *testing.T) {
	if s := Compute(nil, now); s != (Summary{}) {
		t.Errorf("Compute(nil) = %+v, want zero", s)
	}
}

func TestTodayAndRecent(t *testing.T) {
	s := Compute(commitsAt(daysAgo(0, 9), daysAgo(0, 20), daysAgo(1, 10), daysAgo(200, 10)), now)
	if s.Today != 2 {
		t.Errorf("Today = %d, want 2", s.Today)
	}
	if s.Recent != 3 {
		t.Errorf("Recent = %d, want 3 (the 200-day-old one is outside the window)", s.Recent)
	}
}

func TestTodayUsesLocalMidnight(t *testing.T) {
	// 00:30 in Madrid is still "yesterday" in UTC: it must count as today.
	justAfterMidnight := time.Date(2026, 11, 16, 0, 30, 0, 0, madrid)
	s := Compute(commitsAt(justAfterMidnight.UTC()), now)
	if s.Today != 1 {
		t.Errorf("Today = %d, want 1", s.Today)
	}
}

func TestStreak(t *testing.T) {
	tests := []struct {
		name string
		days []int // days ago with a commit
		want int
	}{
		{"none", nil, 0},
		{"only today", []int{0}, 1},
		{"three in a row", []int{0, 1, 2}, 3},
		{"alive before today's commit", []int{1, 2, 3}, 3},
		{"gap breaks it", []int{0, 1, 3, 4}, 2},
		{"broken two days ago", []int{2, 3}, 0},
		{"several commits a day count once", []int{0, 0, 1, 1}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var times []time.Time
			for _, d := range tt.days {
				times = append(times, daysAgo(d, 12))
			}
			if got := Compute(commitsAt(times...), now).Streak; got != tt.want {
				t.Errorf("Streak = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGrowthCurve(t *testing.T) {
	one := Compute(commitsAt(now), now).Growth
	if want := 1 - math.Exp(-1/Scale); math.Abs(one-want) > 1e-9 {
		t.Errorf("one fresh commit: growth = %v, want %v", one, want)
	}

	// A commit exactly one half-life old counts half.
	old := Compute(commitsAt(now.Add(-HalfLife)), now).Growth
	if want := 1 - math.Exp(-0.5/Scale); math.Abs(old-want) > 1e-9 {
		t.Errorf("half-life-old commit: growth = %v, want %v", old, want)
	}

	// Three commits a day for two months keeps the tree lush but not maxed.
	var steady []time.Time
	for d := range 60 {
		for h := range 3 {
			steady = append(steady, daysAgo(d, 10+h))
		}
	}
	g := Compute(commitsAt(steady...), now).Growth
	if g < 0.85 || g >= 1 {
		t.Errorf("steady rhythm: growth = %.3f, want between 0.85 and 1", g)
	}
}

func TestGrowthFadesWithTime(t *testing.T) {
	events := commitsAt(daysAgo(0, 10), daysAgo(0, 11), daysAgo(1, 10))
	fresh := Compute(events, now).Growth
	later := Compute(events, now.AddDate(0, 0, 30)).Growth
	if !(later < fresh && later > 0) {
		t.Errorf("growth should fade but not vanish: fresh %.3f, a month later %.3f", fresh, later)
	}
}

func TestFocusSessions(t *testing.T) {
	focus := func(at time.Time, minutes float64) store.Event {
		return store.Event{Kind: store.KindFocus, Repo: "pomodoro", Ref: at.String(), At: at, Amount: minutes}
	}
	events := []store.Event{focus(daysAgo(0, 10), 50), focus(daysAgo(0, 15), 25), focus(daysAgo(1, 10), 25)}
	s := Compute(events, now)
	if s.FocusToday != 75 || s.Today != 0 {
		t.Errorf("FocusToday = %d, Today = %d; want 75 and 0", s.FocusToday, s.Today)
	}
	if s.Streak != 2 {
		t.Errorf("focus days should keep the streak: %d", s.Streak)
	}
	// A fresh 25-minute session is worth exactly one fresh commit.
	pomo := Compute([]store.Event{focus(now, 25)}, now).Growth
	commit := Compute(commitsAt(now), now).Growth
	if math.Abs(pomo-commit) > 1e-9 {
		t.Errorf("25 min of focus = %v growth, one commit = %v", pomo, commit)
	}
}

func TestZeroAmountCountsAsOne(t *testing.T) {
	e := store.Event{Kind: store.KindCommit, At: now}
	if points(e) != 1 {
		t.Errorf("points = %v, want 1", points(e))
	}
}

func TestFutureEventsIgnored(t *testing.T) {
	s := Compute(commitsAt(now.Add(time.Hour)), now)
	if s.Recent != 0 || s.Growth != 0 {
		t.Errorf("future event counted: %+v", s)
	}
}

func TestPrevAcrossMonthAndDST(t *testing.T) {
	if got := (civilDay{2026, time.March, 1}).prev(); got != (civilDay{2026, time.February, 28}) {
		t.Errorf("prev of Mar 1 = %+v", got)
	}
	// Europe/Madrid switches to summer time on 2026-03-29.
	if got := (civilDay{2026, time.March, 30}).prev(); got != (civilDay{2026, time.March, 29}) {
		t.Errorf("prev across DST = %+v", got)
	}
}

func TestActiveTime(t *testing.T) {
	active := func(at time.Time, minutes float64) store.Event {
		return store.Event{Kind: store.KindActive, Repo: "computer", Ref: at.String(), At: at, Amount: minutes}
	}
	events := []store.Event{active(daysAgo(0, 9), 60), active(daysAgo(0, 10), 35), active(daysAgo(1, 9), 20)}
	s := Compute(events, now)
	if s.ActiveToday != 95 || s.Today != 0 || s.FocusToday != 0 {
		t.Errorf("ActiveToday = %d, Today = %d, FocusToday = %d; want 95, 0, 0", s.ActiveToday, s.Today, s.FocusToday)
	}
	if s.Streak != 2 {
		t.Errorf("days at the computer keep the streak: %d", s.Streak)
	}
	// A fresh hour at the computer is worth exactly one fresh commit.
	hour := Compute([]store.Event{active(now, 60)}, now).Growth
	commit := Compute(commitsAt(now), now).Growth
	if math.Abs(hour-commit) > 1e-9 {
		t.Errorf("an hour active = %v growth, one commit = %v", hour, commit)
	}
}

func TestClock(t *testing.T) {
	for minutes, want := range map[int]string{0: "0m", 45: "45m", 60: "1h", 65: "1h05", 130: "2h10", 600: "10h"} {
		if got := Clock(minutes); got != want {
			t.Errorf("Clock(%d) = %q, want %q", minutes, got, want)
		}
	}
}
