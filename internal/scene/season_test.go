package scene

import (
	"testing"
	"time"
)

func TestSeasonOf(t *testing.T) {
	day := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 12, 0, 0, 0, time.UTC) }
	tests := []struct {
		at   time.Time
		want Season
	}{
		{day(time.January, 15), Winter},
		{day(time.February, 28), Winter},
		{day(time.March, 1), Spring}, // first day of each season
		{day(time.May, 31), Spring},  // and last
		{day(time.June, 1), Summer},
		{day(time.August, 31), Summer},
		{day(time.September, 1), Autumn},
		{day(time.November, 17), Autumn},
		{day(time.December, 1), Winter},
		{day(time.December, 31), Winter},
	}
	for _, tt := range tests {
		if got := SeasonOf(tt.at); got != tt.want {
			t.Errorf("SeasonOf(%s) = %s, want %s", tt.at.Format("Jan 2"), got, tt.want)
		}
	}
}

func TestParseSeasonRoundTrip(t *testing.T) {
	for _, s := range Seasons {
		got, ok := ParseSeason(s.String())
		if !ok || got != s {
			t.Errorf("ParseSeason(%q) = %v, %v", s.String(), got, ok)
		}
	}
	if _, ok := ParseSeason("monsoon"); ok {
		t.Error("unknown season should not parse")
	}
	if _, ok := ParseSeason("Autumn"); ok {
		t.Error("season names are lowercase")
	}
}

func TestSeasonDefaults(t *testing.T) {
	if (State{}).Season != Summer {
		t.Error("the zero State should be summer, the classic look")
	}
	if Season(42).String() != "Season(42)" {
		t.Errorf("unknown season String() = %q", Season(42).String())
	}
	if len(Seasons) != 4 || Seasons[0] != Spring {
		t.Errorf("Seasons = %v", Seasons)
	}
}
