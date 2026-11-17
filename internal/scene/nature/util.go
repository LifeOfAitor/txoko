package nature

import (
	"math"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
)

// Reveal draws the part of tree t that growth g (0..1) shows, dressed for
// the season. Every percent of growth reveals the same amount of tree, and
// growing only ever adds to the picture. It returns the visible marks and,
// among them, the leaves still on the tree (for falling weather and
// fireflies). dx shifts the tree sideways on the canvas.
func Reveal(c *canvas.Canvas, t Tree, g float64, season scene.Season, dx int) (visible, leaves []Mark) {
	visible = t.Marks[:int(math.Round(Clamp01(g)*float64(len(t.Marks))))]
	for _, m := range visible {
		if !m.Leaf {
			c.Set(m.X+dx, m.Y, m.R, m.Ink)
			continue
		}
		r, ink, onTree := DressLeaf(m, season)
		if onTree {
			c.Set(m.X+dx, m.Y, r, ink)
			leaves = append(leaves, Mark{X: m.X + dx, Y: m.Y, R: r, Ink: ink, Leaf: true})
		}
	}
	return visible, leaves
}

// DrawFireflies lets a few fireflies hover around the leaves on warm
// nights, never below the floor row.
func DrawFireflies(c *canvas.Canvas, floor int, leaves []Mark, g float64, frame int, season scene.Season) {
	warm := season == scene.Summer || season == scene.Spring
	if !warm || len(leaves) == 0 || g < 0.3 {
		return
	}
	for i := range 1 + int(g*3) {
		home := leaves[(i*7919+frame/80)%len(leaves)]
		x := home.X + Triangle(frame/2+i*5, 7) - 3
		y := home.Y + 1 + Triangle(frame/3+i*3, 3) - 1
		glyph := '•'
		if (frame/2+i)%5 == 0 {
			glyph = '·'
		}
		if y <= floor {
			c.SetIfEmpty(x, y, glyph, canvas.InkFirefly)
		}
	}
}

// Triangle maps an ever-increasing t to a value bouncing between 0 and n-1.
func Triangle(t, n int) int {
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

// Clamp01 limits v to 0..1, treating NaN as 0.
func Clamp01(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return math.Min(1, math.Max(0, v))
}
