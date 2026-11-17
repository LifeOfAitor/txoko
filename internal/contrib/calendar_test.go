package contrib

import (
	"testing"
	"time"
)

func day(s string, n int) Day {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return Day{Date: d, Count: n}
}

// sample is two weeks: Sunday 2 Nov 2025 to Saturday 15 Nov 2025, plus the
// Sunday after.
func sample() Calendar {
	counts := []int{0, 3, 1, 0, 2, 2, 2, 5, 0, 0, 1, 1, 1, 1, 4}
	var c Calendar
	start := day("2025-11-02", 0).Date
	for i, n := range counts {
		c.Days = append(c.Days, Day{Date: start.AddDate(0, 0, i), Count: n})
	}
	return c
}

func TestTotalAndStreak(t *testing.T) {
	c := sample()
	if got := c.Total(); got != 23 {
		t.Errorf("Total() = %d, want 23", got)
	}
	if got := c.LongestStreak(); got != 5 { // 12 to 16 Nov
		t.Errorf("LongestStreak() = %d, want 5", got)
	}
	gap := Calendar{Days: []Day{day("2025-11-01", 1), day("2025-11-02", 1), day("2025-11-05", 1)}}
	if got := gap.LongestStreak(); got != 2 {
		t.Errorf("a gap in the dates breaks a streak: got %d, want 2", got)
	}
	if (Calendar{}).LongestStreak() != 0 || (Calendar{}).Total() != 0 {
		t.Error("an empty calendar has nothing")
	}
}

func TestWeeks(t *testing.T) {
	weeks := sample().Weeks()
	want := []Week{
		{day("2025-11-02", 0).Date, 10},
		{day("2025-11-09", 0).Date, 9},
		{day("2025-11-16", 0).Date, 4},
	}
	if len(weeks) != len(want) {
		t.Fatalf("got %d weeks, want %d", len(weeks), len(want))
	}
	for i := range want {
		if !weeks[i].Start.Equal(want[i].Start) || weeks[i].Count != want[i].Count {
			t.Errorf("week %d = %v, want %v", i, weeks[i], want[i])
		}
	}
	// A calendar that starts midweek starts with a partial week.
	c := sample()
	c.Days = c.Days[3:]
	if w := c.Weeks()[0]; !w.Start.Equal(want[0].Start) || w.Count != 6 {
		t.Errorf("first partial week = %v", w)
	}
}

func TestBestWeek(t *testing.T) {
	best, ok := sample().BestWeek()
	if !ok || best.Count != 10 || best.Start.Day() != 2 {
		t.Errorf("BestWeek() = %v, %v", best, ok)
	}
	if _, ok := (Calendar{Days: []Day{day("2025-11-02", 0)}}).BestWeek(); ok {
		t.Error("a calendar with nothing done has no best week")
	}
}

func TestDate(t *testing.T) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Skip("no time zone data:", err)
	}
	late := time.Date(2025, 11, 2, 23, 30, 0, 0, madrid) // still the 2nd in Madrid, the 2nd in UTC too
	early := time.Date(2025, 11, 3, 0, 30, 0, 0, madrid) // the 3rd in Madrid, the 2nd in UTC
	if got := Date(late); !got.Equal(day("2025-11-02", 0).Date) {
		t.Errorf("Date(late) = %v", got)
	}
	if got := Date(early); !got.Equal(day("2025-11-03", 0).Date) {
		t.Errorf("Date(early) = %v: a day is the day where it happened", got)
	}
}

func TestFromTimes(t *testing.T) {
	loc := time.FixedZone("UTC+1", 3600)
	from := time.Date(2025, 11, 1, 12, 0, 0, 0, loc)
	to := time.Date(2025, 11, 5, 9, 0, 0, 0, loc)
	times := []time.Time{
		time.Date(2025, 11, 1, 0, 30, 0, 0, loc),                     // first day, before from's time: still counts
		time.Date(2025, 11, 3, 10, 0, 0, 0, loc),                     // middle
		time.Date(2025, 11, 3, 23, 30, 0, 0, time.UTC),               // the 4th at 00:30 in loc
		time.Date(2025, 11, 5, 23, 0, 0, 0, loc),                     // last day
		time.Date(2025, 10, 31, 12, 0, 0, 0, loc),                    // before: ignored
		time.Date(2025, 11, 6, 0, 0, 0, 0, loc).Add(time.Nanosecond), // after: ignored
	}
	c := FromTimes(times, from, to, loc)
	want := []int{1, 0, 1, 1, 1}
	if len(c.Days) != len(want) {
		t.Fatalf("got %d days, want %d", len(c.Days), len(want))
	}
	for i, n := range want {
		if c.Days[i].Count != n {
			t.Errorf("day %d (%s) = %d, want %d", i, c.Days[i].Date.Format(time.DateOnly), c.Days[i].Count, n)
		}
	}
	if !c.Days[0].Date.Equal(day("2025-11-01", 0).Date) {
		t.Errorf("first day = %v", c.Days[0].Date)
	}
}
