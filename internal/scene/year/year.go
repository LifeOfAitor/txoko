// Package year draws a year of activity as a landscape: one column per
// week, from a year ago on the left to this week on the right. The busier
// the week, the more grew there, from bare earth to grass, flowers, bushes
// and trees, each in the colors of its own season. The month names run
// along the bottom, and the sky is the bonsai's.
package year

import (
	"math"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/contrib"
	"github.com/LifeOfAitor/txoko/internal/scene"
	"github.com/LifeOfAitor/txoko/internal/scene/nature"
)

// State is what changes from one drawing to the next.
type State struct {
	Night     bool
	MoonPhase float64
	Frame     int
}

// What grows on a week, by height in rows.
const (
	grassRows  = 1
	flowerRows = 2
	bushRows   = 4 // up to this tall it's a bush; taller, a tree
)

// layout is where everything goes for a given canvas size.
type layout struct {
	w, h    int
	ground  int // the soil row; the month names go below it
	cw      int // columns per week
	x0      int // where the first week starts
	weeks   []contrib.Week
	heights []int // rows of growth per week
	sky     []int // per canvas column: the highest row with something on it
}

// newLayout needs at least one week; Draw makes sure of that.
func newLayout(w, h int, weeks []contrib.Week) layout {
	l := layout{w: w, h: h, ground: h - 2, weeks: weeks}
	l.cw = min(max(w/len(weeks), 1), 3)
	l.x0 = max(0, (w-l.cw*len(weeks))/2)

	busiest := 0
	for _, wk := range weeks {
		busiest = max(busiest, wk.Count)
	}
	// The busiest week reaches up to a few rows below the top, leaving sky.
	// A square root lets quieter weeks still show: half the busiest week's
	// work grows about 70% as tall.
	tallest := max(1, l.ground-3)
	for _, wk := range weeks {
		height := 0
		if wk.Count > 0 {
			height = max(1, int(math.Round(math.Sqrt(float64(wk.Count)/float64(busiest))*float64(tallest))))
		}
		l.heights = append(l.heights, height)
	}
	l.sky = make([]int, w)
	for x := range l.sky {
		l.sky[x] = l.ground
	}
	for i, height := range l.heights {
		x := l.x(i)
		reach := 0
		if height > bushRows {
			reach = 1 // a tree's crown is wider than its week
		}
		for dx := -reach; dx < l.cw+reach; dx++ {
			if x+dx >= 0 && x+dx < w {
				l.sky[x+dx] = min(l.sky[x+dx], l.ground-height)
			}
		}
	}
	return l
}

// x is the first canvas column of week i.
func (l layout) x(i int) int { return l.x0 + i*l.cw }

// UnderCanopy keeps the sky off the plants and the ground.
func (l layout) UnderCanopy(x, y, margin int) bool {
	if x < 0 || x >= len(l.sky) {
		return y >= l.ground-margin
	}
	return y >= l.sky[x]-margin
}

// Floor is the lowest row of the sky.
func (l layout) Floor() int { return l.ground - 1 }

// MoonFrame hangs the moon over the quietest stretch of the year, where
// the sky is clearest, instead of over this week, which is often the
// tallest. The further right, the better, among equally clear spots.
func (l layout) MoonFrame() (cx, top, maxR int) {
	r := min(max(l.w/14, 2), 5, max(1, l.Floor()/2))
	best, bestClear := -1, 0
	for c := r + 3; c <= l.w-r-3; c++ {
		open := l.ground
		for x := c - r - 3; x <= c+r+2; x++ {
			open = min(open, l.sky[x])
		}
		if open >= bestClear {
			best, bestClear = c, open
		}
	}
	// The disc needs its rows and one more below it free of plants.
	for ; r >= 2; r-- {
		if best >= 0 && bestClear > r+2 {
			return best, 1, r
		}
	}
	return 0, 0, 0
}

// Draw paints the year onto c, replacing whatever was there.
func Draw(c *canvas.Canvas, seed uint64, weeks []contrib.Week, s State) {
	c.Clear()
	w, h := c.Width(), c.Height()
	if w < 4 || h < 4 || len(weeks) == 0 {
		return
	}
	l := newLayout(w, h, weeks)
	drawMonths(c, l)
	for i, wk := range weeks {
		drawWeek(c, l, seed, i, scene.SeasonOf(wk.Start))
	}
	if s.Night {
		nature.DimForNight(c)
		nature.DrawSky(c, seed, l, s.MoonPhase, s.Frame)
	} else {
		nature.DrawClouds(c, seed, []int{1, max(2, l.ground/3)}, s.Frame)
	}
}

// drawWeek grows one week's plants on its own patch of ground, dressed for
// its season.
func drawWeek(c *canvas.Canvas, l layout, seed uint64, i int, season scene.Season) {
	x0, height := l.x(i), l.heights[i]
	for dx := range l.cw {
		x := x0 + dx
		soil, ink := []rune("_._,_")[noise(x, l.ground, seed, 1, 5)], canvas.InkSoil
		if season == scene.Winter {
			soil, ink = []rune{'_', '.', '_'}[x%3], canvas.InkSnow
		}
		c.Set(x, l.ground, soil, ink)
	}
	switch {
	case height == 0:
	case height <= grassRows:
		drawGrass(c, l, seed, x0, season)
	case height <= flowerRows:
		drawFlowers(c, l, seed, x0, season)
	case height <= bushRows:
		drawBush(c, l, seed, x0, height, season)
	default:
		drawTree(c, l, seed, x0, height, season)
	}
}

func drawGrass(c *canvas.Canvas, l layout, seed uint64, x0 int, season scene.Season) {
	for dx := range l.cw {
		x := x0 + dx
		r, ink := []rune(",'\"`")[noise(x, l.ground-1, seed, 2, 4)], canvas.InkGrass
		switch season {
		case scene.Autumn:
			ink = canvas.InkAutumnGold
		case scene.Winter:
			r, ink = '.', canvas.InkSnow
		}
		c.Set(x, l.ground-1, r, ink)
	}
}

func drawFlowers(c *canvas.Canvas, l layout, seed uint64, x0 int, season scene.Season) {
	for dx := range l.cw {
		x := x0 + dx
		stem, bloom, ink := 'ψ', '✿', canvas.InkPetalYellow
		switch season {
		case scene.Spring:
			ink = canvas.InkBlossom
		case scene.Autumn:
			bloom, ink = '*', canvas.InkAutumnOrange
		case scene.Winter:
			stem, bloom, ink = 'ı', '*', canvas.InkSnow
		}
		if noise(x, l.ground, seed, 3, 3) == 0 {
			bloom = stem // now and then just a taller stem
		}
		c.Set(x, l.ground-1, stem, canvas.InkGrass)
		c.Set(x, l.ground-2, bloom, ink)
	}
}

// leaf is a leaf glyph and color for the season.
func leaf(x, y int, seed uint64, season scene.Season) (rune, canvas.Ink) {
	r := []rune("&%@*#")[noise(x, y, seed, 4, 5)]
	v := noise(x, y, seed, 5, 10)
	switch season {
	case scene.Spring:
		if v < 6 {
			return r, canvas.InkBlossom
		}
		return r, canvas.InkBlossomLight
	case scene.Autumn:
		return r, []canvas.Ink{canvas.InkAutumnRed, canvas.InkAutumnOrange, canvas.InkAutumnGold}[v%3]
	case scene.Winter:
		return []rune("*.:")[v%3], canvas.InkSnow
	}
	if v < 4 {
		return r, canvas.InkLeafDark
	}
	return r, canvas.InkLeaf
}

// drawBush grows a rounded clump of leaves as tall as the week allows; in
// winter, a shrub under the snow.
func drawBush(c *canvas.Canvas, l layout, seed uint64, x0, height int, season scene.Season) {
	for dy := 1; dy <= height; dy++ {
		y := l.ground - dy
		for dx := range l.cw {
			if dy == height && l.cw > 2 && (dx == 0 || dx == l.cw-1) {
				continue // round off the top
			}
			r, ink := leaf(x0+dx, y, seed, season)
			if season == scene.Winter && dy < height && noise(x0+dx, y, seed, 6, 4) == 0 {
				r, ink = '|', canvas.InkWood // a twig sticks out of the snow
			}
			c.Set(x0+dx, y, r, ink)
		}
	}
}

// drawTree grows a trunk and a crown as wide as the week plus one column on
// each side.
func drawTree(c *canvas.Canvas, l layout, seed uint64, x0, height int, season scene.Season) {
	mid := x0 + l.cw/2
	crown := min(5, max(2, height/2))
	for dy := 1; dy <= height-crown; dy++ {
		c.Set(mid, l.ground-dy, '|', canvas.InkWood)
	}
	for dy := height - crown + 1; dy <= height; dy++ {
		y := l.ground - dy
		reach := 1
		if dy == height {
			reach = 0 // a rounder top
		}
		for x := x0 - reach; x < x0+l.cw+reach; x++ {
			r, ink := leaf(x, y, seed, season)
			if season == scene.Winter && x == mid && dy < height {
				r, ink = '|', canvas.InkWood // the bare branches show through
			}
			c.Set(x, y, r, ink)
		}
	}
}

// drawMonths writes the month names below the ground, under the week each
// month starts in, as long as they don't run into each other. The first
// week gets its own month's name too.
func drawMonths(c *canvas.Canvas, l layout) {
	y := l.h - 1
	next := 0 // the first column free for a label
	for i, wk := range l.weeks {
		label, ok := wk.Start, i == 0
		for d := range 7 {
			if day := wk.Start.AddDate(0, 0, d); day.Day() == 1 {
				label, ok = day, true
			}
		}
		name, x := label.Format("Jan"), l.x(i)
		if !ok || x < next || x+len(name) > l.w {
			continue
		}
		for k, r := range name {
			c.Set(x+k, y, r, canvas.InkRockDim)
		}
		next = x + len(name) + 1
	}
}

// noise picks one of n choices for a cell, the same every time.
func noise(x, y int, seed uint64, salt uint64, n int) int {
	return min(n-1, int(nature.Noise(x, y, seed*131+salt)*float64(n)))
}
