// Package garden draws txoko's garden scene: plants that sprout, bud and
// bloom as progress grows, with a day/night sky on top.
//
// Drawing is a pure function of (canvas size, seed, State): the same inputs
// always produce the same picture. Randomness comes only from the seed, and
// animation only from State.Frame, which is what makes the scene testable
// with golden files.
package garden

import (
	"math"
	"math/rand/v2"

	"github.com/LifeOfAitor/txoko/internal/canvas"
)

// State is everything that changes the picture besides the seed and size.
type State struct {
	// Growth is overall progress from 0 (bare soil) to 1 (full bloom).
	// Values outside that range are clamped.
	Growth float64
	// Frame is an ever-increasing animation counter.
	Frame int
	// Night switches the sky to moon, stars and fireflies.
	Night bool
}

// Growth thresholds, relative to each plant's own progress.
const (
	budAt   = 0.55
	bloomAt = 0.8
)

// Draw paints the garden onto c, replacing whatever was there.
func Draw(c *canvas.Canvas, seed uint64, s State) {
	c.Clear()
	w, h := c.Width(), c.Height()
	if w == 0 || h == 0 {
		return
	}

	ground := h - 1
	for x := range w {
		c.Set(x, ground, '▔', canvas.InkSoil)
	}
	if h < 3 {
		return
	}

	g := clamp01(s.Growth)
	l := newLayout(seed, w, h)
	base := h - 2 // the row plants stand on

	for _, p := range l.plants {
		p.draw(c, base, g)
	}
	drawGrass(c, l, base, g, s.Frame)

	if s.Night {
		drawNightSky(c, l, base, g, s.Frame)
	} else {
		drawDaySky(c, l, base, g, s.Frame)
	}
}

// layout holds every random decision for a given seed and size, so the
// garden stays put from frame to frame.
type layout struct {
	plants []plant
	grass  []grassTuft
	stars  []star
	phase  [4]int // offsets that de-synchronize animated details
}

type grassTuft struct {
	threshold float64 // tuft appears once growth exceeds this
	glyph     rune
}

type star struct {
	x, y, phase int
}

var grassGlyphs = []rune{',', '\'', '"', '.', '`'}

func newLayout(seed uint64, w, h int) layout {
	rng := rand.New(rand.NewPCG(seed, uint64(w)<<32|uint64(h)))
	base := h - 2
	var l layout

	const spacing = 6
	slots := max(1, (w-2)/spacing)
	for i := range slots {
		x := 1 + i*spacing + 1 + rng.IntN(spacing-2)
		x = min(max(x, 1), w-2)
		avail := base // rows 1..base, row 0 is kept for the sun and moon
		maxH := 0
		if avail > 0 {
			maxH = max(1, avail*(5+rng.IntN(5))/10)
		}
		l.plants = append(l.plants, plant{
			x:     x,
			kind:  kinds[rng.IntN(len(kinds))],
			maxH:  maxH,
			start: rng.Float64() * 0.6,
		})
	}
	// The earliest plant always starts at zero so something sprouts right away.
	first := 0
	for i, p := range l.plants {
		if p.start < l.plants[first].start {
			first = i
		}
	}
	l.plants[first].start = 0

	l.grass = make([]grassTuft, w)
	for x := range l.grass {
		l.grass[x] = grassTuft{
			threshold: rng.Float64(),
			glyph:     grassGlyphs[rng.IntN(len(grassGlyphs))],
		}
	}

	// Stars live in the upper half of the sky, sparse enough to stay calm.
	if sky := base / 2; sky > 0 {
		for range max(1, w*sky/20) {
			l.stars = append(l.stars, star{x: rng.IntN(w), y: rng.IntN(sky), phase: rng.IntN(5)})
		}
	}

	for i := range l.phase {
		l.phase[i] = rng.IntN(997)
	}
	return l
}

func drawGrass(c *canvas.Canvas, l layout, base int, g float64, frame int) {
	if g == 0 {
		return
	}
	density := 0.15 + 0.55*g
	// A gentle gust of wind sweeps across the meadow now and then.
	w := c.Width()
	gust := (frame/2+l.phase[0])%(w+24) - 12
	for x, tuft := range l.grass {
		if tuft.threshold >= density {
			continue
		}
		glyph := tuft.glyph
		if abs(x-gust) <= 1 {
			glyph = '/'
		}
		c.SetIfEmpty(x, base, glyph, canvas.InkGrass)
	}
}

func drawDaySky(c *canvas.Canvas, l layout, base int, g float64, frame int) {
	w := c.Width()
	c.SetIfEmpty(w-3, 0, '☼', canvas.InkSun)

	// One small cloud drifting right to left.
	if base > 3 && w >= 16 {
		const cloud = ".-~~-."
		x := w - (frame/12+l.phase[1])%(w+len(cloud))
		for i, r := range cloud {
			c.SetIfEmpty(x+i, 1, r, canvas.InkCloud)
		}
	}

	// Butterflies show up once the garden is in bloom.
	if g >= 0.6 {
		n := 1 + int((g-0.6)*5)
		for i := range n {
			t := frame/3 + l.phase[2] + i*53
			x := triangle(t, w)
			y := base - 1 - triangle(t/2+i*7, max(1, base/2))
			glyph := 'v'
			if (frame/2+i)%2 == 0 {
				glyph = '^'
			}
			c.SetIfEmpty(x, y, glyph, canvas.InkButterfly)
		}
	}
}

func drawNightSky(c *canvas.Canvas, l layout, base int, g float64, frame int) {
	w := c.Width()
	c.SetIfEmpty(w-3, 0, '☾', canvas.InkMoon)

	for _, s := range l.stars {
		glyph := '·'
		if (frame/6+s.phase)%5 == 0 {
			glyph = '✦'
		}
		c.SetIfEmpty(s.x, s.y, glyph, canvas.InkStar)
	}

	// Fireflies drift low over the plants and blink.
	n := int(g * 5)
	for i := range n {
		t := frame/3 + l.phase[3] + i*37
		x := triangle(t, w)
		y := base - 1 - triangle(t/2+i*11, max(1, base/2))
		glyph := '•'
		if (frame/2+i)%5 == 0 {
			glyph = '·'
		}
		c.SetIfEmpty(x, y, glyph, canvas.InkFirefly)
	}
}

// triangle maps an ever-increasing t to a value bouncing between 0 and n-1.
func triangle(t, n int) int {
	if n <= 1 {
		return 0
	}
	period := 2 * (n - 1)
	m := t % period
	if m < 0 {
		m += period
	}
	if m < n {
		return m
	}
	return period - m
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return math.Min(1, math.Max(0, v))
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
