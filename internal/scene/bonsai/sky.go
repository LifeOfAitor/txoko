package bonsai

import (
	"math"
	"math/rand/v2"

	"github.com/LifeOfAitor/txoko/internal/canvas"
)

// drawSky paints the night sky: the moon in its real phase, stars of four
// sizes that twinkle at their own pace, and now and then a shooting star.
// Everything stays above the tree's silhouette.
func drawSky(c *canvas.Canvas, seed uint64, t tree, moonPhase float64, frame int) {
	moon := drawMoon(c, t, moonPhase)
	drawStars(c, seed, t, moon, frame)
	drawShootingStar(c, seed, t, frame)
}

// box is a rectangle of cells.
type box struct{ x0, y0, x1, y1 int }

func (b box) contains(x, y, margin int) bool {
	return x >= b.x0-margin && x <= b.x1+margin && y >= b.y0-margin && y <= b.y1+margin
}

// drawMoon draws a pixel-art moon in the top right corner and returns the
// box it covers. Each terminal cell holds two square "pixels" stacked
// vertically (the ▀ and ▄ half blocks), so the disc comes out round. The
// sunlit part is solid; the rest is a faint texture, so the whole disc is
// always visible, like earthshine on a real night.
func drawMoon(c *canvas.Canvas, t tree, phase float64) box {
	w := c.Width()
	// The disc is r rows tall and 2r cells wide. It grows with the screen,
	// stays within the upper half of the sky, and shrinks if it would touch
	// the crown.
	r := min(max(w/12, 2), 6, max(1, t.baseY/2))
	place := func(r int) box {
		x0, y0 := w-3-2*r, 0
		if t.baseY > 3*r {
			y0 = 1 // a little breathing room when the sky is tall
		}
		return box{x0, y0, x0 + 2*r - 1, y0 + r - 1}
	}
	touches := func(b box) bool {
		for x := b.x0 - 2; x <= b.x1+2; x++ {
			if t.underCanopy(x, b.y1, 1) {
				return true
			}
		}
		return false
	}
	for r > 2 && touches(place(r)) {
		r--
	}
	if w < 20 || r < 2 {
		c.SetIfEmpty(w-3, 0, '☾', canvas.InkMoon)
		return box{w - 3, 0, w - 3, 0}
	}
	b := place(r)
	cols, rows, x0, y0 := 2*r, r, b.x0, b.y0

	k := math.Cos(2 * math.Pi * phase)
	lit := func(u, v float64) bool {
		edge := math.Sqrt(math.Max(0, 1-v*v))
		if phase < 0.5 {
			return u >= k*edge // waxing: lit from the right
		}
		return u <= -k*edge // waning: lit from the left
	}
	for cy := range rows {
		for cx := range cols {
			u := (float64(cx) + 0.5 - float64(cols)/2) / (float64(cols) / 2)
			vTop := (float64(2*cy) + 0.5 - float64(rows)) / float64(rows)
			vBot := (float64(2*cy+1) + 0.5 - float64(rows)) / float64(rows)
			inTop, inBot := u*u+vTop*vTop <= 1, u*u+vBot*vBot <= 1
			litTop, litBot := inTop && lit(u, vTop), inBot && lit(u, vBot)

			var glyph rune
			ink := canvas.InkMoon
			switch {
			case litTop && litBot:
				glyph = '█'
			case litTop:
				glyph = '▀'
			case litBot:
				glyph = '▄'
			case inTop || inBot:
				glyph, ink = '░', canvas.InkMoonDark
			default:
				continue
			}
			c.SetIfEmpty(x0+cx, y0+cy, glyph, ink)
		}
	}
	return b
}

// Star sizes, from the faintest to the brightest.
const (
	tiny = iota
	small
	medium
	big
)

// drawStars scatters stars over the open sky. Most are tiny and faint, a
// few are bright, and the biggest sparkle with little rays.
func drawStars(c *canvas.Canvas, seed uint64, t tree, moon box, frame int) {
	w := c.Width()
	rows := max(2, t.baseY*3/4) // keep the lowest sky, near the pot, clear
	rng := rand.New(rand.NewPCG(seed, uint64(w)<<20|uint64(c.Height())^0x57a75))

	count := max(3, w*rows/12)
	for i := range count {
		x, y, phase := rng.IntN(w), rng.IntN(rows), rng.IntN(60)
		size := tiny
		switch v := rng.Float64(); {
		case i == 0 && w >= 40, v > 0.97:
			size = big // a wide sky always gets at least one
		case v > 0.85:
			size = medium
		case v > 0.6:
			size = small
		}
		if t.underCanopy(x, y, 1) || moon.contains(x, y, 1) {
			continue
		}
		if size == big && (t.underCanopy(x-1, y, 1) || t.underCanopy(x+1, y, 1) || t.underCanopy(x, y+1, 1)) {
			size = medium // its rays would reach into the tree
		}
		drawStar(c, x, y, size, phase, frame)
	}
}

func drawStar(c *canvas.Canvas, x, y, size, phase, frame int) {
	switch size {
	case tiny:
		if (frame/9+phase)%13 != 0 { // now and then a faint one blinks out
			c.SetIfEmpty(x, y, '·', canvas.InkStarDim)
		}
	case small:
		glyph := '·'
		if (frame/7+phase)%6 == 0 {
			glyph = '+'
		}
		c.SetIfEmpty(x, y, glyph, canvas.InkStar)
	case medium:
		glyph := []rune("✦✦✧+")[(frame/8+phase)%4]
		c.SetIfEmpty(x, y, glyph, canvas.InkStarBright)
	case big:
		center := '✦'
		if (frame/6+phase)%4 == 0 {
			center = '✧'
		}
		c.SetIfEmpty(x, y, center, canvas.InkStarBright)
		if (frame/10+phase)%3 != 0 { // the rays pulse
			for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				c.SetIfEmpty(x+d[0], y+d[1], '·', canvas.InkStar)
			}
		}
	}
}

// Shooting stars: one crosses the sky every shootingEvery frames and is
// visible for shootingFor frames.
const (
	shootingEvery = 360
	shootingFor   = 14
)

func drawShootingStar(c *canvas.Canvas, seed uint64, t tree, frame int) {
	step := frame % shootingEvery
	if step >= shootingFor {
		return
	}
	cycle := uint64(frame / shootingEvery)
	rng := rand.New(rand.NewPCG(seed, cycle^0x5ad))
	w := c.Width()
	startX := w/2 + rng.IntN(max(1, w/2))
	startY := rng.IntN(max(1, t.baseY/3))

	// It falls down and to the left, leaving a short fading trail.
	headX, headY := startX-2*step, startY+step/2
	for i, glyph := range []rune{'✦', '·', '·', '.'} {
		x, y := headX+2*i, headY-i/2
		if t.underCanopy(x, y, 0) {
			continue
		}
		ink := canvas.InkStar
		if i == 0 {
			ink = canvas.InkStarBright
		}
		c.SetIfEmpty(x, y, glyph, ink)
	}
}
