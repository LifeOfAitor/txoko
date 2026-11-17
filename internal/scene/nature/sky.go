package nature

import (
	"math"
	"math/rand/v2"

	"github.com/LifeOfAitor/txoko/internal/canvas"
)

// Canopy is the solid part of a scene that the sky stays out of: a tree, an
// island, a wall.
type Canopy interface {
	// UnderCanopy reports whether (x, y) is at or below the silhouette,
	// widened by margin rows.
	UnderCanopy(x, y, margin int) bool
	// Floor is the lowest row of the sky.
	Floor() int
}

// MoonFramer is an optional extra for a Canopy: a scene that wants the moon
// somewhere other than the top right corner, say framed in an arch, tells
// where. MoonFrame returns the column the disc is centered on, its top row
// and the biggest radius that fits there; maxR < 2 means no room, and the
// moon goes back to its corner. A framed moon hangs far behind the scene:
// whatever the scene drew in front of it, a tree say, hides part of it.
type MoonFramer interface {
	MoonFrame() (cx, top, maxR int)
}

// DrawSky paints the night sky: the moon in the given phase, stars of four
// sizes that twinkle at their own pace, and now and then a shooting star.
// Everything stays above the canopy's silhouette.
func DrawSky(c *canvas.Canvas, seed uint64, t Canopy, moonPhase float64, frame int) {
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
func drawMoon(c *canvas.Canvas, t Canopy, phase float64) box {
	w := c.Width()
	// The disc is r rows tall and 2r cells wide. It grows with the screen,
	// stays within the upper half of the sky, and shrinks if it would touch
	// the crown.
	r := min(max(w/12, 2), 6, max(1, t.Floor()/2))
	place := func(r int) box {
		x0, y0 := w-3-2*r, 0
		if t.Floor() > 3*r {
			y0 = 1 // a little breathing room when the sky is tall
		}
		return box{x0, y0, x0 + 2*r - 1, y0 + r - 1}
	}
	touches := func(b box) bool {
		for x := b.x0 - 2; x <= b.x1+2; x++ {
			if t.UnderCanopy(x, b.y1, 1) {
				return true
			}
		}
		return false
	}
	framed := false
	if f, ok := t.(MoonFramer); ok {
		if cx, top, maxR := f.MoonFrame(); maxR >= 2 {
			r, framed = min(maxR, 6), true
			place = func(r int) box { return box{cx - r, top, cx + r - 1, top + r - 1} }
		}
	}
	for !framed && r > 2 && touches(place(r)) {
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
func drawStars(c *canvas.Canvas, seed uint64, t Canopy, moon box, frame int) {
	w := c.Width()
	rows := max(2, t.Floor()*3/4) // keep the lowest sky, near the pot, clear
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
		if t.UnderCanopy(x, y, 1) || moon.contains(x, y, 1) {
			continue
		}
		if size == big && (t.UnderCanopy(x-1, y, 1) || t.UnderCanopy(x+1, y, 1) || t.UnderCanopy(x, y+1, 1)) {
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

// Shooting stars: one crosses the sky every ShootingEvery frames and is
// visible for ShootingFor frames.
const (
	ShootingEvery = 360
	ShootingFor   = 14
)

func drawShootingStar(c *canvas.Canvas, seed uint64, t Canopy, frame int) {
	step := frame % ShootingEvery
	if step >= ShootingFor {
		return
	}
	cycle := uint64(frame / ShootingEvery)
	rng := rand.New(rand.NewPCG(seed, cycle^0x5ad))
	w := c.Width()
	startX := w/2 + rng.IntN(max(1, w/2))
	startY := rng.IntN(max(1, t.Floor()/3))

	// It falls down and to the left, leaving a short fading trail.
	headX, headY := startX-2*step, startY+step/2
	for i, glyph := range []rune{'✦', '·', '·', '.'} {
		x, y := headX+2*i, headY-i/2
		if t.UnderCanopy(x, y, 0) {
			continue
		}
		ink := canvas.InkStar
		if i == 0 {
			ink = canvas.InkStarBright
		}
		c.SetIfEmpty(x, y, glyph, ink)
	}
}

// DrawClouds lets a cloud drift slowly across each of the given rows by
// day, each at its own pace. They pass behind whatever the scene has drawn
// and, if t isn't nil, behind its whole canopy, gaps and all.
func DrawClouds(c *canvas.Canvas, seed uint64, t Canopy, rows []int, frame int) {
	shapes := []string{".-~~-.", ".-~~~-.", ".-~-."}
	w := c.Width()
	for i, row := range rows {
		if row < 0 || row >= c.Height() {
			continue
		}
		shape := shapes[i%len(shapes)]
		speed := 14 + i*7
		span := w + len(shape)
		x := w - (frame/speed+int(Noise(i, int(seed%997), 7)*float64(span)))%span
		for dx, r := range shape {
			if t == nil || !t.UnderCanopy(x+dx, row, 0) {
				c.SetIfEmpty(x+dx, row, r, canvas.InkCloud)
			}
		}
	}
}
