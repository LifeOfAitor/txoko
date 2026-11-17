// Package island draws a floating island: a chunk of rock adrift in the
// sky, with a tree growing on it, roots dangling below and, once it has
// grown enough, a waterfall pouring off its edge. The tree, the sky and the
// seasons come from package nature, so it shares the bonsai's look.
package island

import (
	"math"

	"github.com/LifeOfAitor/txoko/internal/canvas"
	"github.com/LifeOfAitor/txoko/internal/scene"
	"github.com/LifeOfAitor/txoko/internal/scene/nature"
)

// Growth thresholds for the island's own details.
const (
	waterfallAt = 0.45 // the spring starts flowing
	birdsAt     = 0.7  // birds come to visit
)

// Scene exposes the island through the scene.Scene interface.
type Scene struct{}

// Name implements scene.Scene.
func (Scene) Name() string { return "island" }

// Draw implements scene.Scene.
func (Scene) Draw(c *canvas.Canvas, seed uint64, s scene.State) { Draw(c, seed, s) }

// layout is where everything goes for a given canvas size.
type layout struct {
	w, h     int
	cx       int // island center column
	top      int // the island's surface row
	half     int // half width of the surface
	depth    int // rows of rock below the surface
	tree     nature.Tree
	treeDX   int // the tree's offset on the canvas
	fallX    int // the waterfall's column
	fallLen  int // the waterfall's full length
	tiny     bool
	rockEnds []int // per rock row: half width
}

func newLayout(seed uint64, w, h int) layout {
	l := layout{w: w, h: h, cx: w / 2}
	if w < 16 || h < 8 {
		l.tiny = true
		l.top = h - 3
		l.half = max(2, w/4)
		l.depth = 1
	} else {
		l.top = int(math.Round(float64(h) * 0.6))
		l.half = min(max(w/4, 5), 22)
		l.depth = min(max(l.half/3, 2), 6, max(1, h-l.top-3))
	}
	for r := 1; r <= l.depth; r++ {
		f := math.Pow(1-float64(r)/float64(l.depth+1), 0.85)
		l.rockEnds = append(l.rockEnds, max(1, int(math.Round(float64(l.half)*f))))
	}
	// The tree grows from the middle of the surface. It plans its size from
	// the room it's given, so it gets a little more than the island's width:
	// its crown may lean out over the edges, but it stays in proportion with
	// the island instead of towering over it in a tall pane.
	tw := min(w, 2*l.half+10)
	l.tree = nature.Grow(seed, tw, l.top+2)
	l.treeDX = l.cx - tw/2
	l.fallX = l.cx + l.half + 1
	// The waterfall pours for a while and then vanishes into spray in mid
	// air, rather than running down to the bottom of a tall pane.
	l.fallLen = min(h-2-l.top, max(3, 2*l.depth+2))
	return l
}

// UnderCanopy keeps the sky out of the island, its tree and its waterfall.
func (l layout) UnderCanopy(x, y, margin int) bool {
	if l.tree.UnderCanopy(x-l.treeDX, y, margin) {
		return true
	}
	if y >= l.top-1-margin && y <= l.top+l.depth+1 && abs(x-l.cx) <= l.half+1+margin {
		return true
	}
	return abs(x-l.fallX) <= 1+margin && y >= l.top && y <= l.top+l.fallLen+2
}

// inRock reports whether (x, y) is part of the rock below the surface.
func (l layout) inRock(x, y int) bool {
	r := y - l.top - 1
	return r >= 0 && r < len(l.rockEnds) && abs(x-l.cx) <= l.rockEnds[r]
}

// Floor is the bottom of the canvas: the island floats, so there's sky
// all around it.
func (l layout) Floor() int { return l.h - 1 }

// Draw paints the island onto c, replacing whatever was there.
func Draw(c *canvas.Canvas, seed uint64, s scene.State) {
	l, leaves, g, ok := drawIsland(c, seed, s)
	if !ok {
		return
	}
	drawWaterfall(c, l, g, s)
	if s.Night {
		nature.DimForNight(c)
		nature.DrawSky(c, seed, l, s.MoonPhase, s.Frame)
		nature.DrawFireflies(c, l.top-1, leaves, g, s.Frame, s.Season)
	} else {
		drawClouds(c, l, seed, s.Frame)
		if g >= birdsAt && s.Season != scene.Winter {
			drawBirds(c, l, s.Frame)
		}
	}
	nature.DrawWeather(c, seed, l.Floor(), leaves, g, s.Frame, s.Season)
}

// drawIsland paints everything that doesn't move: rock, soil, roots, grass
// and the tree, dressed for the season. Growing only ever adds to it.
func drawIsland(c *canvas.Canvas, seed uint64, s scene.State) (l layout, leaves []nature.Mark, g float64, ok bool) {
	c.Clear()
	w, h := c.Width(), c.Height()
	if w < 6 || h < 4 {
		return layout{}, nil, 0, false
	}
	l = newLayout(seed, w, h)
	g = nature.Clamp01(s.Growth)

	drawRock(c, l)
	drawRoots(c, l, g)
	visible, leaves := nature.Reveal(c, l.tree, g, s.Season, l.treeDX)
	drawGrass(c, l, g, s.Season)
	if s.Season == scene.Winter {
		nature.SnowOnBranches(c, l.tree, visible, l.treeDX)
	}
	return l, leaves, g, true
}

// drawRock draws the surface and the rock tapering below it:
//
//	\_.~_:__.,_~__.:_/
//	  \:.;:.':.;:./
//	     \:.;:./
//	        \/
func drawRock(c *canvas.Canvas, l layout) {
	soil := []rune("_._:_~_.,_")
	for x := l.cx - l.half + 1; x < l.cx+l.half; x++ {
		r := soil[int(nature.Noise(x, l.top, 3)*float64(len(soil)))]
		ink := canvas.InkSoil
		if r == '_' {
			ink = canvas.InkRock
		}
		c.Set(x, l.top, r, ink)
	}
	c.Set(l.cx-l.half, l.top, '\\', canvas.InkRock)
	c.Set(l.cx+l.half, l.top, '/', canvas.InkRock)

	texture := []rune(":.;:'. :")
	for i, half := range l.rockEnds {
		y := l.top + 1 + i
		for x := l.cx - half + 1; x < l.cx+half; x++ {
			c.Set(x, y, texture[int(nature.Noise(x, y, 5)*float64(len(texture)))], canvas.InkRockDim)
		}
		c.Set(l.cx-half, y, '\\', canvas.InkRock)
		c.Set(l.cx+half, y, '/', canvas.InkRock)
	}
}

// rootCell is one cell of a root. at is the growth at which it shows up,
// so a growing root only ever adds cells at its tip.
type rootCell struct {
	x, y int
	r    rune
	at   float64
}

// roots lays out every root hanging from the rock, fully grown. Each one
// starts right under the rock, wanders down with a slight sway, may fork
// once and thins out towards its tip:
//
//	\:.;:./
//	 |  |\
//	 /  ¦ ¦
//	¦
func (l layout) roots() [][]rootCell {
	if l.tiny || len(l.rockEnds) == 0 {
		return nil
	}
	n := min(max(l.half/6, 2), 4)
	maxLen := 2 + l.depth + l.half/4
	// They hang from the underside, not from the overhangs near the edge.
	span := l.rockEnds[len(l.rockEnds)/2]
	var all [][]rootCell
	for i := range n {
		f := -1 + 2*(float64(i)+0.5)/float64(n) + nature.Noise(i, l.half, 67)*0.3 - 0.15
		x := l.cx + int(math.Round(f*float64(span)))
		y := l.top + 1
		for l.inRock(x, y) {
			y++
		}
		room := l.h - 1 - y
		length := min(room, int(math.Round(float64(maxLen)*(0.45+0.55*nature.Noise(i, l.depth, 63)))))
		if length < 1 {
			continue
		}
		main := l.root(i, x, y, length, 0, 0, 1)
		all = append(all, main)
		// A long root forks once, a third of the way down.
		if fork := length / 3; length >= 8 && fork < len(main) {
			from := main[fork]
			span := 1 - from.at
			side := 1
			if nature.Noise(i, fork, 69) < 0.5 {
				side = -1
			}
			all = append(all, l.root(i+17, from.x, from.y+1, (length-fork)*2/3, side, from.at, span))
		}
	}
	return all
}

// root walks one root down from (x, y). A fork bends away towards side
// for its first two steps; its cells show up between growth start and
// start+span.
func (l layout) root(salt, x, y, length, side int, start, span float64) []rootCell {
	var cells []rootCell
	prev := 0
	for k := range length {
		d := side
		if k >= 2 || side == 0 && k > 0 {
			switch v := nature.Noise(salt*31+k, x, 61); {
			case v < 0.15:
				d = -1
			case v > 0.85:
				d = 1
			default:
				d = 0
			}
			if d == -prev {
				d = 0 // a gentle sway, never a zigzag
			}
		}
		// Roots stay out of the rock and away from the waterfall. Going
		// straight down always works: the rock only narrows below.
		if l.inRock(x+d, y) || abs(x+d-l.fallX) < 2 {
			d = 0
		}
		x += d
		prev = d
		r := '|'
		switch {
		case d < 0:
			r = '/'
		case d > 0:
			r = '\\'
		case k >= length*2/3:
			r = '¦' // thin towards the tip
		}
		cells = append(cells, rootCell{x, y, r, start + span*float64(k+1)/float64(length)})
		y++
	}
	return cells
}

// drawRoots lets the roots dangle below the rock, longer as it grows.
func drawRoots(c *canvas.Canvas, l layout, g float64) {
	for _, root := range l.roots() {
		for _, cell := range root {
			if cell.at <= g {
				c.SetIfEmpty(cell.x, cell.y, cell.r, canvas.InkWoodDim)
			}
		}
	}
}

// drawGrass covers the surface with grass and, by season, flowers, fallen
// leaves or snow. Each spot has its own threshold, so the island greens up
// gradually and never loses a blade it already had.
func drawGrass(c *canvas.Canvas, l layout, g float64, season scene.Season) {
	y := l.top - 1
	blades := []rune(",'`.,'")
	for x := l.cx - l.half + 1; x < l.cx+l.half; x++ {
		// What grows on each spot is decided once and for all; growth only
		// decides when it shows up, so nothing ever changes into something
		// else as the island grows. Flowers and ferns come later than grass.
		appear := nature.Noise(x, y, 41) * 0.9
		glyph, ink := blades[int(nature.Noise(x, y, 43)*float64(len(blades)))], canvas.InkGrass
		switch bloom := nature.Noise(x, y, 47); {
		case season == scene.Winter:
			glyph, ink = []rune{'_', '.', '_'}[x%3], canvas.InkSnow
		case season == scene.Autumn && bloom < 0.3:
			glyph, ink = ',', []canvas.Ink{canvas.InkAutumnRed, canvas.InkAutumnOrange, canvas.InkAutumnGold}[x%3]
		case season == scene.Spring && bloom < 0.18:
			glyph, ink, appear = '✿', canvas.InkBlossom, max(appear, 0.4)
		case season == scene.Summer && bloom < 0.1:
			glyph, ink, appear = '✿', canvas.InkPetalYellow, max(appear, 0.5)
		case bloom > 0.94:
			glyph, appear = 'ψ', max(appear, 0.3)
		}
		if g > appear {
			c.SetIfEmpty(x, y, glyph, ink)
		}
	}
	if season == scene.Winter {
		for x := l.cx - l.half + 1; x < l.cx+l.half; x++ {
			if c.At(x, l.top).Ink == canvas.InkSoil || c.At(x, l.top).Rune == '_' {
				c.Set(x, l.top, []rune{'_', '.', '_'}[x%3], canvas.InkSnow)
			}
		}
	}
}

// drawWaterfall pours water off the island's right edge once the spring
// has started, longer as the island grows. Fully grown it spreads out and
// breaks into spray in mid air. In winter it freezes into icicles.
func drawWaterfall(c *canvas.Canvas, l layout, g float64, s scene.State) {
	if l.tiny || g < waterfallAt || l.fallLen < 1 {
		return
	}
	length := min(l.fallLen, int(math.Ceil((g-waterfallAt)/0.35*float64(l.fallLen))))
	c.Set(l.fallX-1, l.top, '~', canvas.InkWater)
	if s.Season == scene.Winter {
		for k := 1; k <= min(length, 3); k++ {
			c.SetIfEmpty(l.fallX, l.top+k, []rune{'¡', '|', 'i'}[k-1], canvas.InkSnow)
		}
		return
	}
	flow := []rune{'│', '¦', '│', ':'}
	for k := 1; k <= length; k++ {
		y := l.top + k
		ink := canvas.InkWater
		if (k+s.Frame/2)%3 == 0 {
			ink = canvas.InkWaterDim
		}
		c.SetIfEmpty(l.fallX, y, flow[(k-s.Frame/2%4+400)%4], ink)
		// Lower down, drops start flying off to the sides.
		if k > l.fallLen/2 {
			for _, side := range []int{-1, 1} {
				if (k+s.Frame/2+side+2)%3 == 0 {
					c.SetIfEmpty(l.fallX+side, y, []rune{'\'', '.'}[(k+s.Frame)%2], canvas.InkWaterDim)
				}
			}
		}
	}
	if length < l.fallLen {
		return
	}
	// Spray where it vanishes.
	y := l.top + length + 1
	for dx, r := range []rune{'·', '°', ' ', '°', '·'} {
		if r != ' ' && (s.Frame/3+dx)%4 != 0 {
			c.SetIfEmpty(l.fallX-2+dx, y, r, canvas.InkWaterDim)
		}
	}
	for _, dx := range []int{-1, 1} {
		if (s.Frame/4+dx+2)%3 != 0 {
			c.SetIfEmpty(l.fallX+dx*(1+(s.Frame/5)%2), y+1, '·', canvas.InkWaterDim)
		}
	}
}

// drawClouds lets a few clouds drift slowly past, above and below the
// island.
func drawClouds(c *canvas.Canvas, l layout, seed uint64, frame int) {
	shapes := []string{".-~~-.", ".-~~~-.", ".-~-."}
	rows := []int{1, l.top / 3, l.h - 2}
	for i, row := range rows {
		if row < 0 || row >= l.h || l.tiny {
			continue
		}
		shape := shapes[i%len(shapes)]
		speed := 14 + i*7
		span := l.w + len(shape)
		x := l.w - (frame/speed+int(nature.Noise(i, int(seed%997), 7)*float64(span)))%span
		for dx, r := range shape {
			c.SetIfEmpty(x+dx, row, r, canvas.InkCloud)
		}
	}
}

// drawBirds sends two birds gliding in the open sky beside the island.
func drawBirds(c *canvas.Canvas, l layout, frame int) {
	for i := range 2 {
		x := l.w - 1 - nature.Triangle(frame/4+i*11, max(2, l.w/3))
		y := 2 + nature.Triangle(frame/9+i*5, max(2, l.top/3)) + i
		glyph := 'v'
		if (frame/3+i)%2 == 0 {
			glyph = '^'
		}
		if !l.UnderCanopy(x, y, 1) {
			c.SetIfEmpty(x, y, glyph, canvas.InkRockDim)
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
