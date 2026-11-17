package nature

import (
	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
)

// DressLeaf decides how one leaf looks in a season: its glyph and ink, and
// whether it is still on the tree at all. The choice depends only on the
// leaf's position, so every leaf keeps its color from frame to frame and
// growing the tree never recolors what was already there.
func DressLeaf(m Mark, season scene.Season) (rune, canvas.Ink, bool) {
	v := Noise(m.X, m.Y, uint64(season)+1)
	switch season {
	case scene.Spring:
		// Full cherry blossom: two pinks and, here and there, a whole flower.
		r := m.R
		if Noise(m.X, m.Y, 77) < 0.1 {
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
			return m.R, canvas.InkAutumnRed, true
		case v < 0.7:
			return m.R, canvas.InkAutumnOrange, true
		}
		return m.R, canvas.InkAutumnGold, true

	case scene.Winter:
		// Bare in places, the rest of the crown heavy with snow.
		if v < 0.4 {
			return 0, 0, false
		}
		snow := []rune("**.:*")[int(Noise(m.X, m.Y, 91)*5)]
		if v < 0.75 {
			return snow, canvas.InkSnow, true
		}
		return snow, canvas.InkBlossomLight, true
	}
	return m.R, m.Ink, true
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

// DrawWeather animates falling petals, leaves or snow down to the floor row.
func DrawWeather(c *canvas.Canvas, seed uint64, floor int, leaves []Mark, g float64, frame int, season scene.Season) {
	wx := weatherFor(season, c.Width(), c.Height())
	if g < wx.minGrow {
		return
	}
	if !wx.fromTree {
		drawSnowfall(c, seed, floor, wx, frame)
		return
	}
	if len(leaves) == 0 {
		return
	}
	for i := range wx.count {
		f := frame + i*47
		from := leaves[(f/wx.period*31+i*13)%len(leaves)]
		step := f % wx.period
		y := from.Y + 1 + step/4
		if y > floor {
			continue
		}
		x := from.X + Triangle(step/5, 3) - 1
		if season == scene.Spring {
			x += step / 9 // petals ride the breeze sideways
		}
		glyph := wx.glyphs[(step/4)%len(wx.glyphs)]
		c.SetIfEmpty(x, y, glyph, wx.inks[i%len(wx.inks)])
	}
}

// drawSnowfall lets snowflakes drift down the whole sky, each in its own
// lane and at its own pace.
func drawSnowfall(c *canvas.Canvas, seed uint64, floor int, wx weather, frame int) {
	w := c.Width()
	rows := floor + 1
	for i := range wx.count {
		lane := int(Noise(i, int(seed%9973), 11) * float64(w))
		speed := 2 + int(Noise(i, 3, 12)*3) // frames per row: 2, 3 or 4
		offset := int(Noise(i, 5, 13) * float64(rows*speed))
		y := ((frame + offset) / speed) % rows
		x := lane + Triangle((frame+offset)/7, 3) - 1
		c.SetIfEmpty(x, y, wx.glyphs[i%len(wx.glyphs)], wx.inks[i%len(wx.inks)])
	}
}

// Noise returns a stable pseudo-random number in [0, 1) for a cell, so the
// same leaf always gets the same color (splitmix64 finalizer).
func Noise(x, y int, salt uint64) float64 {
	z := uint64(x)*0x9e3779b97f4a7c15 ^ uint64(y)*0xc2b2ae3d27d4eb4f ^ salt*0x165667b19e3779f9
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	return float64(z>>11) / (1 << 53)
}

// SnowOnBranches lets snow settle on top of some branches. It never settles
// where the tree will grow later: growing must only ever add to the
// picture. dx shifts the tree like in Reveal.
func SnowOnBranches(c *canvas.Canvas, t Tree, visible []Mark, dx int) {
	for _, m := range visible {
		if !m.Leaf && Noise(m.X, m.Y, 31) < 0.75 && !t.WillGrow(m.X, m.Y-1) {
			c.SetIfEmpty(m.X+dx, m.Y-1, '.', canvas.InkSnow)
		}
	}
}
