// Package bonsai draws txoko's flagship scene: a bonsai tree in a pot that
// really grows. The tree is simulated tip by tip from a seed — the trunk
// climbs, limbs reach out, twigs fork and burst into clumps of leaves — and
// progress decides how much of that growth you get to see.
package bonsai

import (
	"math"
	"math/rand/v2"
	"sync"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
)

// potHeight is the number of rows the pot takes at the bottom.
const potHeight = 2

// Scene exposes the bonsai through the scene.Scene interface.
type Scene struct{}

// Name implements scene.Scene.
func (Scene) Name() string { return "bonsai" }

// Draw implements scene.Scene.
func (Scene) Draw(c *canvas.Canvas, seed uint64, s scene.State) { Draw(c, seed, s) }

// Draw paints the bonsai onto c, replacing whatever was there.
func Draw(c *canvas.Canvas, seed uint64, s scene.State) {
	c.Clear()
	w, h := c.Width(), c.Height()
	if w == 0 || h == 0 {
		return
	}
	drawPot(c, seed)
	if h <= potHeight {
		return
	}

	t := grown(seed, w, h)
	g := clamp01(s.Growth)
	// Every percent of growth reveals the same amount of tree.
	visible := t.marks[:int(math.Round(g*float64(len(t.marks))))]

	var shown []mark // visible leaves, for the animations
	for _, m := range visible {
		c.Set(m.x, m.y, m.r, m.ink)
		if m.leaf {
			shown = append(shown, m)
		}
	}

	if s.Night {
		drawNight(c, seed, t, shown, g, s.Frame)
	}
	if g >= 0.6 {
		drawFallingLeaves(c, t, shown, s.Frame)
	}
}

// drawPot draws a shallow pot centered at the bottom of the canvas:
//
//	\_.~_:__.,_~_/
//	 \__________/
func drawPot(c *canvas.Canvas, seed uint64) {
	w, h := c.Width(), c.Height()
	pw := min(max(w*2/5, 7), 36, w)
	x0 := (w - pw) / 2
	rng := rand.New(rand.NewPCG(seed, 0x9075))

	soil := []rune("___.__:_~__.,_")
	rim := h - potHeight
	for x := x0 + 1; x < x0+pw-1; x++ {
		r := soil[rng.IntN(len(soil))]
		ink := canvas.InkPot
		if r != '_' {
			ink = canvas.InkSoil
		}
		c.Set(x, rim, r, ink)
	}
	c.Set(x0, rim, '\\', canvas.InkPot)
	c.Set(x0+pw-1, rim, '/', canvas.InkPot)

	base := h - 1
	for x := x0 + 2; x < x0+pw-2; x++ {
		c.Set(x, base, '_', canvas.InkPot)
	}
	c.Set(x0+1, base, '\\', canvas.InkPot)
	c.Set(x0+pw-2, base, '/', canvas.InkPot)
}

// drawNight adds a moon, a few twinkling stars and fireflies that hover
// around the leaves.
func drawNight(c *canvas.Canvas, seed uint64, t tree, leaves []mark, g float64, frame int) {
	w := c.Width()
	c.SetIfEmpty(w-3, 0, '☾', canvas.InkMoon)

	rng := rand.New(rand.NewPCG(seed, 0x57a75))
	skyRows := max(1, t.baseY/3)
	for range max(1, w*skyRows/40) {
		x, y, phase := rng.IntN(w), rng.IntN(skyRows), rng.IntN(5)
		glyph := '·'
		if (frame/6+phase)%5 == 0 {
			glyph = '✦'
		}
		c.SetIfEmpty(x, y, glyph, canvas.InkStar)
	}

	if len(leaves) == 0 || g < 0.3 {
		return
	}
	for i := range 1 + int(g*3) {
		home := leaves[(i*7919+frame/80)%len(leaves)]
		x := home.x + triangle(frame/2+i*5, 7) - 3
		y := home.y + 1 + triangle(frame/3+i*3, 3) - 1
		glyph := '•'
		if (frame/2+i)%5 == 0 {
			glyph = '·'
		}
		if y <= t.baseY {
			c.SetIfEmpty(x, y, glyph, canvas.InkFirefly)
		}
	}
}

// drawFallingLeaves lets a couple of leaves drift down from the crown.
func drawFallingLeaves(c *canvas.Canvas, t tree, leaves []mark, frame int) {
	if len(leaves) == 0 {
		return
	}
	const period = 90
	for i := range 2 {
		f := frame + i*47
		from := leaves[(f/period*31+i*13)%len(leaves)]
		step := f % period
		y := from.y + 1 + step/4
		if y > t.baseY {
			continue
		}
		x := from.x + triangle(step/5, 3) - 1
		glyph := ','
		if (step/4)%2 == 1 {
			glyph = '`'
		}
		c.SetIfEmpty(x, y, glyph, canvas.InkLeafDim)
	}
}

// The simulation only depends on seed and size, and the TUI redraws the
// same tree many times per second, so the last result is cached.
var cache struct {
	sync.Mutex
	seed uint64
	w, h int
	tree *tree
}

func grown(seed uint64, w, h int) tree {
	cache.Lock()
	defer cache.Unlock()
	if cache.tree == nil || cache.seed != seed || cache.w != w || cache.h != h {
		t := simulate(seed, w, h)
		cache.seed, cache.w, cache.h, cache.tree = seed, w, h, &t
	}
	return *cache.tree
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
