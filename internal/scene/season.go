package scene

import (
	"fmt"
	"time"
)

// Season changes how a scene dresses itself: blossoms in spring, deep
// green in summer, warm colors in autumn, snow in winter.
type Season uint8

// Seasons, in calendar order. Summer is the zero value, so a State that
// doesn't set a season looks the way scenes always have.
const (
	Summer Season = iota
	Autumn
	Winter
	Spring
)

// Seasons lists every season starting with spring, the order people expect.
var Seasons = []Season{Spring, Summer, Autumn, Winter}

var seasonNames = map[Season]string{
	Spring: "spring",
	Summer: "summer",
	Autumn: "autumn",
	Winter: "winter",
}

func (s Season) String() string {
	if name, ok := seasonNames[s]; ok {
		return name
	}
	return fmt.Sprintf("Season(%d)", uint8(s))
}

// ParseSeason turns a name such as "autumn" into a Season.
func ParseSeason(name string) (Season, bool) {
	for s, n := range seasonNames {
		if n == name {
			return s, true
		}
	}
	return Summer, false
}

// SeasonOf returns the meteorological season of t in the northern
// hemisphere: spring is March to May, summer June to August, autumn
// September to November and winter December to February.
func SeasonOf(t time.Time) Season {
	switch t.Month() {
	case time.March, time.April, time.May:
		return Spring
	case time.June, time.July, time.August:
		return Summer
	case time.September, time.October, time.November:
		return Autumn
	default:
		return Winter
	}
}
