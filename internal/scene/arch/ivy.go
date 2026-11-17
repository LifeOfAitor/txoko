package arch

import (
	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
	"github.com/LifeOfAitor/txoko/internal/scene/nature"
)

// ivyCell is one cell of ivy: a piece of stem or a leaf. at is the growth
// at which it shows up.
type ivyCell struct {
	r    rune
	leaf bool
	at   float64
}

// vine is where one ivy plant starts and how it climbs.
type vine struct {
	x     int
	start float64 // the growth at which it sprouts
	hug   int     // for the vines framing the arch: the side the arch is on
	reach float64 // how much of the wall's height it climbs
}

// vines says where the ivy plants grow. Two frame the arch, climbing its
// jambs and hugging its curve; a few more climb the wall on either side,
// sprouting later.
func (l layout) vines(seed uint64) []vine {
	vines := []vine{
		{x: l.cx - l.aw - 1, start: 0, hug: 1, reach: 1},
		{x: l.cx + l.aw + 1, start: 0.08, hug: -1, reach: 1},
	}
	side := l.cx - l.aw - 3 - l.x0 // wall columns on each side of the arch
	n := min(side/14, 3)
	for i := range n {
		f := (float64(i) + 0.5 + nature.Noise(i, int(seed%89), 91)*0.4 - 0.2) / float64(n)
		dx := 2 + int(f*float64(side-4))
		start := 0.2 + 0.15*float64(i)
		reach := 0.55 + 0.45*nature.Noise(i, int(seed%97), 93)
		vines = append(vines,
			vine{x: l.cx - l.aw - 1 - dx, start: start, reach: reach},
			vine{x: l.cx + l.aw + 1 + dx, start: start + 0.07, reach: 1.55 - reach})
	}
	return vines
}

// grow lays out all the ivy, fully grown, with leaves on both sides of the
// stems. Each cell belongs to one plant and shows up once, so the ivy only
// ever spreads.
func (l layout) grow(seed uint64) map[point]ivyCell {
	stems := map[point]ivyCell{}
	var leaves []struct {
		p point
		c ivyCell
	}
	for i, v := range l.vines(seed) {
		path := l.climb(i, v)
		for k, p := range path {
			at := v.start + (1-v.start)*float64(k+1)/float64(len(path))
			if old, ok := stems[p.point]; !ok || at < old.at {
				stems[p.point] = ivyCell{r: p.r, at: at}
			}
			// Leaves on both sides of the stem, a moment after it.
			for _, d := range []int{-1, 1} {
				if nature.Noise(p.x*3+d, p.y, 95) < 0.6 {
					leaf := ivyCell{r: []rune("&%*@")[int(nature.Noise(p.x+d, p.y, 97)*4)], leaf: true, at: min(1, at+0.04)}
					leaves = append(leaves, struct {
						p point
						c ivyCell
					}{point{p.x + d, p.y}, leaf})
				}
			}
		}
	}
	all := make(map[point]ivyCell, len(stems)+len(leaves))
	for p, c := range stems {
		all[p] = c
	}
	// Leaves only grow on plain stone and never cover a stem; where two
	// would grow, the first one to come out stays.
	for _, lf := range leaves {
		_, onStem := stems[lf.p]
		if onStem || !l.wall(lf.p.x, lf.p.y) || l.ring(lf.p.x, lf.p.y) {
			continue
		}
		if old, ok := all[lf.p]; !ok || lf.c.at < old.at {
			all[lf.p] = lf.c
		}
	}
	return all
}

type step struct {
	point
	r rune
}

// climb walks one vine up the wall from its foot. It sways gently, never
// zigzags, and stays on plain stone. A vine framing the arch leans in
// towards it whenever it can, so it follows the jamb and then the curve.
func (l layout) climb(salt int, v vine) []step {
	// Every vine starts at the foot of the wall, on plain stone: beside the
	// jambs or further out, never in the arch itself.
	x, y := v.x, l.meadow
	height := max(1, int(float64(y-l.top[x])*v.reach))
	path := []step{{point{x, y}, ')'}}
	prev := 0
	for k := 1; k < height; k++ {
		var tries []int
		if v.hug != 0 {
			tries = []int{v.hug, 0, -v.hug}
		} else {
			d := 0
			switch n := nature.Noise(salt*37+k, x, 99); {
			case n < 0.18:
				d = -1
			case n > 0.82:
				d = 1
			}
			if d == -prev {
				d = 0
			}
			tries = []int{d, 0}
		}
		moved := false
		for _, d := range tries {
			if nx := x + d; l.wall(nx, y-1) && !l.ring(nx, y-1) {
				x, y, prev, moved = nx, y-1, d, true
				break
			}
		}
		if !moved {
			break
		}
		// Wavy stems, so they never read as the stone's joints.
		r := []rune{'(', ')'}[(y+salt)%2]
		switch prev {
		case -1:
			r = '('
		case 1:
			r = ')'
		}
		path = append(path, step{point{x, y}, r})
	}
	return path
}

// hasIvy reports whether ivy ever grows on (x, y).
func (l layout) hasIvy(x, y int) bool {
	_, ok := l.ivy[point{x, y}]
	return ok
}

// drawIvy shows the ivy that growth g has reached, dressed for the season:
// green leaves, red in autumn, bare stems under the snow in winter.
func drawIvy(c *canvas.Canvas, l layout, g float64, season scene.Season) {
	for p, cell := range l.ivy {
		if cell.at > g {
			continue
		}
		if !cell.leaf {
			c.Set(p.x, p.y, cell.r, canvas.InkLeafDim)
			continue
		}
		if ink, ok := ivyInk(p, season); ok {
			c.Set(p.x, p.y, cell.r, ink)
		}
	}
}

// ivyInk is a leaf's color in each season; ok is false when it has fallen.
func ivyInk(p point, season scene.Season) (canvas.Ink, bool) {
	v := nature.Noise(p.x, p.y, 101)
	switch season {
	case scene.Spring:
		if v < 0.5 {
			return canvas.InkLeaf, true // fresh new leaves
		}
		return canvas.InkGrass, true
	case scene.Autumn:
		switch {
		case v < 0.55:
			return canvas.InkAutumnRed, true
		case v < 0.85:
			return canvas.InkAutumnOrange, true
		}
		return canvas.InkAutumnGold, true
	case scene.Winter:
		return 0, false
	}
	if v < 0.6 {
		return canvas.InkLeafDark, true
	}
	return canvas.InkLeaf, true
}
