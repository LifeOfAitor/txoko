package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/LifeOfAitor/txoko/internal/contrib"
)

func calendar(source string, counts ...int) contrib.Calendar {
	c := contrib.Calendar{Source: source, Fetched: time.Unix(1_763_380_800, 0)}
	start := time.Date(2025, 11, 2, 0, 0, 0, 0, time.UTC)
	for i, n := range counts {
		c.Days = append(c.Days, contrib.Day{Date: start.AddDate(0, 0, i), Count: n})
	}
	return c
}

func TestCalendarRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	if _, ok, err := s.LoadCalendar(ctx, "github:@me"); ok || err != nil {
		t.Fatalf("nothing saved yet: ok=%v err=%v", ok, err)
	}

	want := calendar("@octo on GitHub", 0, 3, 1)
	if err := s.SaveCalendar(ctx, "github:@me", want); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveCalendar(ctx, "github:other", calendar("@other on GitHub", 9)); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.LoadCalendar(ctx, "github:@me")
	if !ok || err != nil || got.Source != want.Source || !got.Fetched.Equal(want.Fetched) || len(got.Days) != 3 {
		t.Fatalf("got %+v, %v, %v", got, ok, err)
	}
	for i := range want.Days {
		if !got.Days[i].Date.Equal(want.Days[i].Date) || got.Days[i].Count != want.Days[i].Count {
			t.Errorf("day %d = %+v, want %+v", i, got.Days[i], want.Days[i])
		}
	}

	// Saving again replaces the whole calendar, and only that one.
	if err := s.SaveCalendar(ctx, "github:@me", calendar("@octo on GitHub", 5)); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := s.LoadCalendar(ctx, "github:@me"); len(got.Days) != 1 || got.Total() != 5 {
		t.Errorf("after replacing: %+v", got)
	}
	if other, _, _ := s.LoadCalendar(ctx, "github:other"); other.Total() != 9 {
		t.Errorf("another calendar changed: %+v", other)
	}
}

// TestCalendarStorageErrors breaks the database in every way a save or a
// load can notice.
func TestCalendarStorageErrors(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name   string
		damage string
		save   bool
		want   string
	}{
		{"calendars table gone", "DROP TABLE calendars", true, "saving calendar"},
		{"days table gone", "DROP TABLE calendar_days", true, "saving calendar"},
		{"days refused", "CREATE TRIGGER no BEFORE INSERT ON calendar_days BEGIN SELECT RAISE(ABORT, 'no'); END", true, "saving calendar"},
		{"days unreadable", "DROP TABLE calendar_days", false, "reading calendar"},
		{"calendars unreadable", "DROP TABLE calendars", false, "reading calendar"},
		{"bad day", "UPDATE calendar_days SET day = 'someday'", false, "bad day"},
		{"bad count", "UPDATE calendar_days SET count = 'lots'", false, "count"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := open(t)
			if !tt.save {
				if err := s.SaveCalendar(ctx, "k", calendar("x", 1)); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.db.Exec(tt.damage); err != nil {
				t.Fatal(err)
			}
			var err error
			if tt.save {
				err = s.SaveCalendar(ctx, "k", calendar("x", 1))
			} else {
				_, _, err = s.LoadCalendar(ctx, "k")
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("got %v, want an error about %q", err, tt.want)
			}
		})
	}

	s := open(t)
	_ = s.Close()
	if err := s.SaveCalendar(ctx, "k", calendar("x")); err == nil {
		t.Error("saving to a closed database is an error")
	}
}
