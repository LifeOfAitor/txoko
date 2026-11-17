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
	// The tree grows from the middle of the surface with the whole width to
	// spread its crown, which may lean out over the edges.
	l.tree = nature.Grow(seed, w, l.top+2)
	l.treeDX = l.cx - w/2
	l.fallX = l.cx + l.half + 1
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
	return abs(x-l.fallX) <= margin && y >= l.top
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

// drawRoots lets a few roots dangle below the rock, longer as it grows.
func drawRoots(c *canvas.Canvas, l layout, g float64) {
	if l.tiny || len(l.rockEnds) == 0 {
		return
	}
	bottom := l.top + l.depth
	half := l.rockEnds[len(l.rockEnds)-1]
	room := l.h - 1 - bottom
	for i, x := range []int{l.cx - half - 1, l.cx - half/2, l.cx + half/3, l.cx + half} {
		length := int(math.Round(g * float64(room) * (0.35 + 0.15*float64(i%3))))
		for k := 1; k <= length; k++ {
			// The same glyph all along: a longer root only adds to it.
			c.SetIfEmpty(x+(k/3)%2, bottom+k, '¦', canvas.InkWoodDim)
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
// has started, longer as the island grows. In winter it freezes into
// icicles.
func drawWaterfall(c *canvas.Canvas, l layout, g float64, s scene.State) {
	if l.tiny || g < waterfallAt {
		return
	}
	reach := l.h - 1 - l.top
	length := min(reach, int(math.Ceil((g-waterfallAt)/0.35*float64(reach))))
	c.Set(l.fallX-1, l.top, '~', canvas.InkWater)
	if s.Season == scene.Winter {
		for k := 1; k <= min(length, 3); k++ {
			c.SetIfEmpty(l.fallX, l.top+k, []rune{'¡', '|', 'i'}[k-1], canvas.InkSnow)
		}
		return
	}
	flow := []rune{'│', '¦', '│', ':'}
	for k := 1; k <= length; k++ {
		ink := canvas.InkWater
		if (k+s.Frame/2)%3 == 0 {
			ink = canvas.InkWaterDim
		}
		c.SetIfEmpty(l.fallX, l.top+k, flow[(k-s.Frame/2%4+400)%4], ink)
	}
	if length == reach { // spray where it vanishes into the clouds below
		for dx, r := range []rune{'·', '°', ' ', '°', '·'} {
			if r != ' ' && (s.Frame/3+dx)%4 != 0 {
				c.SetIfEmpty(l.fallX-2+dx, l.h-1, r, canvas.InkWaterDim)
			}
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
