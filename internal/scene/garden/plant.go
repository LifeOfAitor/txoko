package garden

import (
	"math"

	"github.com/LifeOfAitor/txoko/internal/canvas"
)

type kind uint8

const (
	daisy kind = iota
	tulip
	sunflower
	fern
)

var kinds = []kind{daisy, tulip, sunflower, fern}

type plant struct {
	x     int
	kind  kind
	maxH  int     // height when fully grown
	start float64 // overall growth at which this plant starts sprouting
}

// progress is the plant's own growth (0..1) for an overall growth g.
// Plants start at different moments, so the garden fills in gradually.
func (p plant) progress(g float64) float64 {
	if g <= p.start {
		return 0
	}
	return clamp01((g - p.start) / (1 - p.start))
}

func (p plant) height(g float64) int {
	return int(math.Ceil(p.progress(g) * float64(p.maxH)))
}

// draw grows the plant upward from row base.
//
//	 ✿      bloom (or 'o' while budding)
//	 |/     leaves alternate sides
//	\|
//	 |
func (p plant) draw(c *canvas.Canvas, base int, g float64) {
	h := p.height(g)
	if h == 0 {
		return
	}
	if h == 1 {
		c.Set(p.x, base, ',', canvas.InkLeaf) // a tiny sprout
		return
	}

	for k := range h {
		y := base - k
		c.Set(p.x, y, '|', canvas.InkStem)
		if k == 0 || k == h-1 {
			continue
		}
		switch {
		case p.kind == fern:
			c.Set(p.x-1, y, '\\', canvas.InkLeaf)
			c.Set(p.x+1, y, '/', canvas.InkLeaf)
		case k%2 == 1 && (k/2)%2 == 0:
			c.Set(p.x-1, y, '\\', canvas.InkLeaf)
		case k%2 == 1:
			c.Set(p.x+1, y, '/', canvas.InkLeaf)
		}
	}

	top := base - (h - 1)
	pg := p.progress(g)
	switch {
	case p.kind == fern:
		c.Set(p.x, top, 'ψ', canvas.InkLeaf) // a fern frond
	case pg >= bloomAt:
		p.drawBloom(c, top)
	case pg >= budAt:
		c.Set(p.x, top, 'o', canvas.InkBud)
	}
}

func (p plant) drawBloom(c *canvas.Canvas, y int) {
	switch p.kind {
	case daisy:
		c.Set(p.x, y, '✿', canvas.InkPetalPink)
	case tulip:
		c.Set(p.x, y, '❀', canvas.InkPetalRed)
	case sunflower:
		c.Set(p.x-1, y, '*', canvas.InkPetalYellow)
		c.Set(p.x, y, '@', canvas.InkCenter)
		c.Set(p.x+1, y, '*', canvas.InkPetalYellow)
	}
}
