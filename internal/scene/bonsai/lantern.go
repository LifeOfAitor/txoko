package bonsai

import (
	"math"

	"github.com/LifeOfAitor/txoko/internal/canvas"
)

// lantern is the warm light that makes a night scene cozy.
type lantern struct {
	x, y   int  // the lantern itself
	hanger bool // hangs from a branch on a short cord (else sits on the soil)
	radius float64
}

// placeLantern finds where the lantern goes. Once the tree has grown the
// chosen branch, the lantern hangs from it; before that it sits on the
// soil beside the trunk. The branch is picked from the fully grown tree, so
// the spot never wanders as the tree grows.
func placeLantern(t tree, shown int, w int) lantern {
	radius := math.Max(3, math.Min(float64(w)/8, float64(t.baseY+1)/3))
	best, bestScore := -1, 2.0
	for i, m := range t.marks {
		if m.leaf || abs(m.x-t.baseX) < 4 || m.y+2 > t.baseY {
			continue
		}
		if t.willGrow(m.x, m.y+1) || t.willGrow(m.x, m.y+2) {
			continue // the cord and lantern need open air below the branch
		}
		if score := noise(m.x, m.y, 55); score < bestScore {
			best, bestScore = i, score
		}
	}
	if best >= 0 && best < shown {
		m := t.marks[best]
		return lantern{x: m.x, y: m.y + 2, hanger: true, radius: radius}
	}
	x := t.baseX + 3
	if t.willGrow(x, t.baseY) {
		x = t.baseX - 3
	}
	return lantern{x: x, y: t.baseY, radius: radius}
}

// light draws the lantern, tints what is right next to it with its warm
// glow and puts everything beyond its reach in shadow. Call it after the
// tree and before the sky, so the moon, the stars and whatever is falling
// through the air keep their own light.
func (l lantern) light(c *canvas.Canvas) {
	for y := range c.Height() {
		for x := range c.Width() {
			if c.IsEmpty(x, y) {
				continue
			}
			dx, dy := float64(x-l.x)/2, float64(y-l.y) // cells are twice as tall as wide
			switch d := math.Hypot(dx, dy); {
			case d > l.radius:
				c.Shade(x, y)
			case d <= l.radius*0.45:
				c.Set(x, y, c.At(x, y).Rune, canvas.InkGlow)
			}
		}
	}
	if l.hanger {
		c.Set(l.x, l.y-1, ':', canvas.InkWoodDim)
	}
	c.Set(l.x, l.y, '¤', canvas.InkLantern)
}
