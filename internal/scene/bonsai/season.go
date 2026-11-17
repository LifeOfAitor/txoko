package bonsai

import (
	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
)

// dressLeaf decides how one leaf looks in a season: its glyph and ink, and
// whether it is still on the tree at all. The choice depends only on the
// leaf's position, so every leaf keeps its color from frame to frame and
// growing the tree never recolors what was already there.
func dressLeaf(m mark, season scene.Season) (rune, canvas.Ink, bool) {
	v := noise(m.x, m.y, uint64(season)+1)
	switch season {
	case scene.Spring:
		// Full cherry blossom: two pinks and, here and there, a whole flower.
		r := m.r
		if noise(m.x, m.y, 77) < 0.1 {
			r = '✿'
		}
		if v < 0.65 {
			return r, canvas.InkBlossom, true
		}
		return r, canvas.InkBlossomLight, true

	case scene.Autumn:
		switch {
		case v < 0.1:
			return 0, 0, false // already fallen
		case v < 0.4:
			return m.r, canvas.InkAutumnRed, true
		case v < 0.7:
			return m.r, canvas.InkAutumnOrange, true
		}
		return m.r, canvas.InkAutumnGold, true

	case scene.Winter:
		// Bare in places, the rest of the crown heavy with snow.
		if v < 0.4 {
			return 0, 0, false
		}
		snow := []rune("**.:*")[int(noise(m.x, m.y, 91)*5)]
		if v < 0.75 {
			return snow, canvas.InkSnow, true
		}
		return snow, canvas.InkBlossomLight, true
	}
	return m.r, m.ink, true
}

// weather is what falls through the air in each season.
type weather struct {
	count    int     // how many drift at once
	minGrow  float64 // growth needed before anything falls
	period   int     // frames for one fall
	glyphs   []rune  // alternate while falling
	inks     []canvas.Ink
	fromTree bool // falls from the leaves (otherwise from the sky)
}

func weatherFor(season scene.Season, w, h int) weather {
	switch season {
	case scene.Spring:
		return weather{9, 0.3, 110, []rune{'.', ',', '*'}, []canvas.Ink{canvas.InkBlossom, canvas.InkBlossomLight}, true}
	case scene.Autumn:
		return weather{9, 0.3, 80, []rune{',', '`'}, []canvas.Ink{canvas.InkAutumnRed, canvas.InkAutumnOrange, canvas.InkAutumnGold}, true}
	case scene.Winter:
		return weather{max(3, w*h/45), 0, 0, []rune{'·', '*', '.'}, []canvas.Ink{canvas.InkSnow, canvas.InkStar}, false}
	}
	return weather{2, 0.6, 90, []rune{',', '`'}, []canvas.Ink{canvas.InkLeafDim}, true}
}

// drawWeather animates falling petals, leaves or snow.
func drawWeather(c *canvas.Canvas, seed uint64, t tree, leaves []mark, g float64, frame int, season scene.Season) {
	wx := weatherFor(season, c.Width(), c.Height())
	if g < wx.minGrow {
		return
	}
	if !wx.fromTree {
		drawSnowfall(c, seed, t, wx, frame)
		return
	}
	if len(leaves) == 0 {
		return
	}
	for i := range wx.count {
		f := frame + i*47
		from := leaves[(f/wx.period*31+i*13)%len(leaves)]
		step := f % wx.period
		y := from.y + 1 + step/4
		if y > t.baseY {
			continue
		}
		x := from.x + triangle(step/5, 3) - 1
		if season == scene.Spring {
			x += step / 9 // petals ride the breeze sideways
		}
		glyph := wx.glyphs[(step/4)%len(wx.glyphs)]
		c.SetIfEmpty(x, y, glyph, wx.inks[i%len(wx.inks)])
	}
}

// drawSnowfall lets snowflakes drift down the whole sky, each in its own
// lane and at its own pace.
func drawSnowfall(c *canvas.Canvas, seed uint64, t tree, wx weather, frame int) {
	w := c.Width()
	rows := t.baseY + 1
	for i := range wx.count {
		lane := int(noise(i, int(seed%9973), 11) * float64(w))
		speed := 2 + int(noise(i, 3, 12)*3) // frames per row: 2, 3 or 4
		offset := int(noise(i, 5, 13) * float64(rows*speed))
		y := ((frame + offset) / speed) % rows
		x := lane + triangle((frame+offset)/7, 3) - 1
		c.SetIfEmpty(x, y, wx.glyphs[i%len(wx.glyphs)], wx.inks[i%len(wx.inks)])
	}
}

// dressGround adds what a season leaves around the tree: fallen autumn
// leaves on the soil, snow on the soil and on top of the branches.
func dressGround(c *canvas.Canvas, t tree, visible []mark, season scene.Season) {
	rim := c.Height() - potHeight
	switch season {
	case scene.Autumn:
		for x := range c.Width() {
			if cell := c.At(x, rim); cell.Ink == canvas.InkSoil || cell.Ink == canvas.InkPot && cell.Rune == '_' {
				if v := noise(x, rim, 21); v < 0.35 {
					inks := []canvas.Ink{canvas.InkAutumnRed, canvas.InkAutumnOrange, canvas.InkAutumnGold}
					c.Set(x, rim, ',', inks[int(v*100)%len(inks)])
				}
			}
		}
	case scene.Winter:
		for x := range c.Width() {
			if cell := c.At(x, rim); cell.Ink == canvas.InkSoil || cell.Ink == canvas.InkPot && cell.Rune == '_' {
				c.Set(x, rim, []rune{'_', '.', '_'}[x%3], canvas.InkSnow)
			}
		}
		// Snow settles on top of some branches, but never where the tree
		// will grow later: growing must only ever add to the picture.
		for _, m := range visible {
			if !m.leaf && noise(m.x, m.y, 31) < 0.75 && !t.willGrow(m.x, m.y-1) {
				c.SetIfEmpty(m.x, m.y-1, '.', canvas.InkSnow)
			}
		}
	}
}

// noise returns a stable pseudo-random number in [0, 1) for a cell, so the
// same leaf always gets the same color (splitmix64 finalizer).
func noise(x, y int, salt uint64) float64 {
	z := uint64(x)*0x9e3779b97f4a7c15 ^ uint64(y)*0xc2b2ae3d27d4eb4f ^ salt*0x165667b19e3779f9
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	return float64(z>>11) / (1 << 53)
}
